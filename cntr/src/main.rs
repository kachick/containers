use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use std::process::Command;
use std::thread;
use std::time::Duration;

#[derive(Parser)]
#[command(name = "cntr")]
#[command(about = "Container management tool for nix-containers", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Build a local image and run it
    Build {
        /// Image name (corresponds to directory in images/)
        #[arg(default_value = "ubuntu-nix-systemd")]
        image: String,

        /// Username for the container
        #[arg(long, default_value = "user")]
        user: String,
    },
    /// Pull an image from a registry and run it
    Pull {
        /// Tag to pull (e.g., latest, pr-42)
        #[arg(default_value = "latest")]
        tag: String,

        /// Image path (registry + package)
        #[arg(long, short, default_value = "ghcr.io/kachick/ubuntu-24.04-nix-systemd")]
        image: String,

        /// Username for the container
        #[arg(long, default_value = "user")]
        user: String,
    },
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match &cli.command {
        Commands::Build { image, user } => {
            run_build(image, user)?;
        }
        Commands::Pull { tag, image, user } => {
            run_pull(tag, image, user)?;
        }
    }

    Ok(())
}

fn run_build(image: &str, user: &str) -> Result<()> {
    println!("Building image: {} for user: {}", image, user);

    let status = Command::new("podman")
        .args([
            "build",
            "--tag",
            image,
            "--build-arg",
            &format!("username={}", user),
            "--file",
            &format!("./images/{}/Containerfile", image),
            ".",
        ])
        .status()
        .context("Failed to execute podman build")?;

    if !status.success() {
        anyhow::bail!("podman build failed");
    }

    start_and_enter_container(image, user)
}

fn run_pull(tag: &str, image_path: &str, user: &str) -> Result<()> {
    let full_image = format!("{}:{}", image_path, tag);
    println!("Pulling image: {}", full_image);

    let status = Command::new("podman")
        .args(["pull", &full_image])
        .status()
        .context("Failed to execute podman pull")?;

    if !status.success() {
        anyhow::bail!("podman pull failed");
    }

    start_and_enter_container(&full_image, user)
}

fn start_and_enter_container(image_id: &str, user: &str) -> Result<()> {
    println!("Starting container...");
    let output = Command::new("podman")
        .args(["run", "--rm", "--detach", image_id])
        .output()
        .context("Failed to execute podman run")?;

    if !output.status.success() {
        anyhow::bail!("podman run failed: {}", String::from_utf8_lossy(&output.stderr));
    }

    let container_id = String::from_utf8_lossy(&output.stdout).trim().to_string();
    println!("Container started: {}", container_id);

    // Wait for systemd to be ready
    thread::sleep(Duration::from_secs(1));

    println!("Entering container shell...");
    let status = Command::new("podman")
        .args([
            "exec",
            "--user",
            user,
            "--interactive",
            "--tty",
            &container_id,
            "bash",
        ])
        .status()
        .context("Failed to execute podman exec")?;

    println!("Stopping container...");
    let _ = Command::new("podman")
        .args(["stop", &container_id])
        .status();

    if !status.success() {
        anyhow::bail!("podman exec session failed");
    }

    Ok(())
}

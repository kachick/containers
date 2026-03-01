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
    /// Build and run the container interactively (replaces try.bash)
    Try {
        /// Username for the container
        #[arg(default_value = "user")]
        user: String,

        /// Image tag name
        #[arg(long, default_value = "ubuntu-nix-systemd")]
        image: String,
    },
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match &cli.command {
        Commands::Try { user, image } => {
            run_try(user, image)?;
        }
    }

    Ok(())
}

fn run_try(user: &str, image: &str) -> Result<()> {
    println!("Building image: {} for user: {}", image, user);

    // 1. podman build
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

    // 2. podman run --detach
    println!("Starting container...");
    let output = Command::new("podman")
        .args(["run", "--rm", "--detach", image])
        .output()
        .context("Failed to execute podman run")?;

    if !output.status.success() {
        anyhow::bail!(
            "podman run failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
    }

    let container_id = String::from_utf8_lossy(&output.stdout).trim().to_string();
    println!("Container started: {}", container_id);

    // 3. sleep 1
    thread::sleep(Duration::from_secs(1));

    // 4. podman exec -it
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

    // 5. podman stop
    println!("Stopping container...");
    let _ = Command::new("podman")
        .args(["stop", &container_id])
        .status();

    if !status.success() {
        anyhow::bail!("podman exec session failed");
    }

    Ok(())
}

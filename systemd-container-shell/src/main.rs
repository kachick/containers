use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use std::process::Command;
use std::thread;
use std::time::Duration;

#[derive(Parser)]
#[command(name = "systemd-container-shell")]
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
        /// Image reference or tag.
        /// If starts with ':', it's a tag for the default image (e.g., :latest, :20260301-094713-utc).
        /// If contains '/', it's treated as a full image path.
        /// Otherwise, it's treated as a tag for the default image for convenience.
        #[arg(default_value = ":latest")]
        reference: String,

        /// Override default image path (registry + package)
        #[arg(long, short, default_value = "ghcr.io/kachick/ubuntu-24.04-nix-systemd")]
        image: String,

        /// Username for the container
        #[arg(long, default_value = "user")]
        user: String,

        /// Skip pulling and use local image if it exists, fail if not found
        #[arg(long)]
        skip_pull: bool,
    },
}

struct ContainerGuard {
    id: String,
}

impl Drop for ContainerGuard {
    fn drop(&mut self) {
        if !self.id.is_empty() {
            println!("Stopping container {}...", self.id);
            let _ = Command::new("podman")
                .args(["stop", &self.id])
                .status();
        }
    }
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match &cli.command {
        Commands::Build { image, user } => {
            run_build(image, user)?;
        }
        Commands::Pull { reference, image, user, skip_pull } => {
            run_pull(reference, image, user, *skip_pull)?;
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

fn run_pull(reference: &str, default_image: &str, user: &str, skip_pull: bool) -> Result<()> {
    let full_image = if reference.starts_with(':') {
        format!("{}{}", default_image, reference)
    } else if reference.contains('/') {
        if reference.contains(':') {
            reference.to_string()
        } else {
            format!("{}:latest", reference)
        }
    } else {
        // For convenience in this repo, treat as tag even without leading ':'
        format!("{}:{}", default_image, reference)
    };

    if skip_pull {
        if image_exists(&full_image)? {
            println!("Image {} found locally, skipping pull", full_image);
        } else {
            anyhow::bail!("Image {} not found locally and --skip-pull specified", full_image);
        }
    } else {
        println!("Pulling image: {}", full_image);

        let status = Command::new("podman")
            .args(["pull", &full_image])
            .status()
            .context("Failed to execute podman pull")?;

        if !status.success() {
            anyhow::bail!("podman pull failed");
        }
    }

    start_and_enter_container(&full_image, user)
}

fn image_exists(image: &str) -> Result<bool> {
    let status = Command::new("podman")
        .args(["image", "exists", image])
        .status()
        .context("Failed to check if image exists")?;
    Ok(status.success())
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

    // RAII: Ensure the container is stopped when this function exits (even on error or panic)
    let _guard = ContainerGuard {
        id: container_id.clone(),
    };

    // Wait for systemd to be ready
    thread::sleep(Duration::from_secs(1));

    println!("Entering container shell...");
    let status = Command::new("podman")
        .args([
            "exec",
            "--user",
            user,
            "--workdir",
            &format!("/home/{}", user),
            "--interactive",
            "--tty",
            &container_id,
            "bash",
        ])
        .status()
        .context("Failed to execute podman exec")?;

    if !status.success() {
        anyhow::bail!("podman exec session failed");
    }

    Ok(())
}

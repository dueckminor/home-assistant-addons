import subprocess
import os
from typing import Optional

root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))


def get_components(component_selector: Optional[str] = None) -> list[str]:
    """Get the list of components to build."""
    all_components = ["gateway", "security", "mqtt-bridge", "alphaess"]
    if component_selector:
        if component_selector in all_components:
            return [component_selector]
        else:
            raise ValueError(f"Unknown component: {component_selector}")
    return all_components


def build_web(component_name: str, fast: bool = False) -> None:
    """Build the web frontend."""

    web_path = os.path.join(root_dir, "web", component_name)

    print(f"🌐 Building web frontend for '{component_name}'...", flush=True)
    print(f"   📂 Path: {web_path}", flush=True)

    if fast:
        dist_path = os.path.join(
            root_dir,
            "go",
            "embed",
            component_name.replace("-", "_") + "_dist",
            "dist",
            "index.html",
        )
        if os.path.isfile(dist_path):
            print(
                f"   ⚡ Fast build enabled and assets already exist at '{dist_path}', skipping build.",
                flush=True,
            )
            return

    print(f"   📦 Installing dependencies...", flush=True)
    subprocess.run(["npm", "install"], cwd=web_path, check=True)

    print(f"   🔨 Building production bundle...", flush=True)
    subprocess.run(["npm", "run", "build"], cwd=web_path, check=True)

    print(f"   ✅ Web frontend build complete!", flush=True)


def build_go(component_name: str) -> None:
    """Build the Go backend binaries for multiple architectures."""

    go_file = os.path.join(
        root_dir, "go", "tools", component_name, f"{component_name}.go"
    )
    output_dir = os.path.join(root_dir, "addons", component_name)
    os.makedirs(output_dir, exist_ok=True)

    architectures = {
        "amd64": "linux/amd64",
        "arm64": "linux/arm64",
    }

    print(f"🔧 Building Go backend for '{component_name}'...", flush=True)
    print(f"   📂 Source: {go_file}", flush=True)
    print(f"   📂 Output: {output_dir}", flush=True)

    for arch, goos_goarch in architectures.items():
        output_file = os.path.join(output_dir, f"{component_name}-{arch}")
        env = os.environ.copy()
        env["CGO_ENABLED"] = "0"
        env["GOOS"], env["GOARCH"] = goos_goarch.split("/")

        print(f"   🏗️  Building for {arch} ({goos_goarch})...", flush=True)
        subprocess.run(["go", "build", "-o", output_file, go_file], env=env, check=True)
        print(f"      ✓ {os.path.basename(output_file)}", flush=True)

    print(f"   ✅ Go backend build complete!", flush=True)


def prepare_local(component_name: str) -> None:
    """Generate local build configuration file."""

    gen_dir = os.path.join(root_dir, "gen", "addons", component_name)
    os.makedirs(gen_dir, exist_ok=True)

    build_yml_path = os.path.join(gen_dir, "build.yml")
    print(f"📝 Generating local build configuration at '{build_yml_path}'...")

    with open(build_yml_path, "w") as f:
        f.write(
            f"""# Local development build configuration
build_from:
  aarch64: "ghcr.io/hassio-addons/base:16.3.6"
  amd64: "ghcr.io/hassio-addons/base:16.3.6"
"""
        )
    print(f"   ✅ Local build configuration generated!")

    # copy all (but not build.yml) files from root_dir/addons/component_name to gen_dir
    src_dir = os.path.join(root_dir, "addons", component_name)
    print(f"📋 Copying addon files from '{src_dir}' to '{gen_dir}'...")
    for item in os.listdir(src_dir):
        if item == "build.yml":
            continue
        src_path = os.path.join(src_dir, item)
        dest_path = os.path.join(gen_dir, item)
        if os.path.isdir(src_path):
            subprocess.run(["cp", "-r", src_path, dest_path], check=True)
        else:
            subprocess.run(["cp", src_path, dest_path], check=True)
    print(f"   ✅ Addon files copied!")


def upload(component_name: str, homeassistant: str = "") -> None:
    """Upload built addon to Home Assistant instance."""

    if not homeassistant:
        homeassistant = "homeassistant.local"
        # check for gen/ha.txt
        ha_txt_path = os.path.join(root_dir, "gen", "ha.txt")
        if os.path.isfile(ha_txt_path):
            with open(ha_txt_path, "r") as f:
                homeassistant = f.read().strip()

    gen_dir = os.path.join(root_dir, "gen", "addons", component_name)

    print(
        f"🚀 Uploading '{component_name}' addon to Home Assistant at '{homeassistant}'..."
    )

    tar_process = subprocess.Popen(
        ["tar", "czf", "-", "."], cwd=gen_dir, stdout=subprocess.PIPE
    )

    component_install_dir = "/addons/dueckminor_" + component_name

    if homeassistant == "localhost":
        cmd = ["bash", "-c"]
        sudo = ""
    else:
        cmd = ["ssh", f"hassio@{homeassistant}"]
        sudo = "sudo "

    cmd.append(
        f"""
            {sudo}rm -rf {component_install_dir}/ &&
            {sudo}mkdir {component_install_dir} &&
            cd {component_install_dir} &&
            {sudo}tar xzvf - &&
            {sudo}chown -R root:root .
            """
    )

    subprocess.run(cmd, stdin=tar_process.stdout, check=True)
    tar_process.stdout.close()
    tar_process.wait()

    print(f"   ✅ Upload complete!")


def build(
    component_name: str,
    additional_web_components: list = [],
    fast: bool = False,
) -> None:
    """Build the addon components."""
    for web_component in additional_web_components:
        build_web(web_component, fast=fast)
    build_web(component_name, fast=fast)
    build_go(component_name)


def install(
    component_name: str, homeassistant: str = "", additional_web_components: list = []
) -> None:
    """Build and upload the addon to Home Assistant instance."""
    for web_component in additional_web_components:
        build_web(web_component)
    build_web(component_name)
    build_go(component_name)
    prepare_local(component_name)
    upload(component_name, homeassistant)

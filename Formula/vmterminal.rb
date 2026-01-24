# Formula/vmterminal.rb
# Homebrew formula for VMTerminal
#
# To install from local tap:
#   brew tap javanstorm/vmterminal https://github.com/javanstorm/vmterminal
#   brew install vmterminal
#
# Or install directly:
#   brew install javanstorm/vmterminal/vmterminal

class Vmterminal < Formula
  desc "Linux VM as your terminal - seamless Linux shell on macOS/Linux"
  homepage "https://github.com/javanstorm/vmterminal"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/javanstorm/vmterminal/releases/download/v#{version}/vmterminal_v#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    else
      url "https://github.com/javanstorm/vmterminal/releases/download/v#{version}/vmterminal_v#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/javanstorm/vmterminal/releases/download/v#{version}/vmterminal_v#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    else
      url "https://github.com/javanstorm/vmterminal/releases/download/v#{version}/vmterminal_v#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "vmterminal"
    # Create aliases
    bin.install_symlink "vmterminal" => "vmt"
    bin.install_symlink "vmterminal" => "vmterm"
  end

  def caveats
    <<~EOS
      VMTerminal requires:
      - macOS 12+ with Virtualization.framework (macOS)
      - KVM support with /dev/kvm access (Linux)

      First run will download a Linux distribution (~50-200MB).

      To set as default terminal shell:
        vmterminal run  # then answer 'y' to the prompt

      Or manually add to your shell profile:
        if [ -z "$VMTERMINAL_SKIP" ] && [ -x $(which vmterminal) ]; then
          exec vmterminal
        fi

      To skip VMTerminal temporarily:
        VMTERMINAL_SKIP=1 zsh
    EOS
  end

  test do
    assert_match "vmterminal", shell_output("#{bin}/vmterminal version")
  end
end

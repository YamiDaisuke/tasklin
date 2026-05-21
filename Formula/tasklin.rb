# This file is managed by GoReleaser and updated automatically on each release.
# Do not edit by hand.
#
# Installation:
#   brew tap yamidaisuke/tasklin https://github.com/yamidaisuke/tasklin
#   brew install tasklin
class Tasklin < Formula
  desc "Keyboard-driven CLI/TUI kanban board for personal project backlogs"
  homepage "https://github.com/yamidaisuke/tasklin"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/yamidaisuke/tasklin/releases/download/v0.0.0/tasklin_darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end

    on_intel do
      url "https://github.com/yamidaisuke/tasklin/releases/download/v0.0.0/tasklin_darwin_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/yamidaisuke/tasklin/releases/download/v0.0.0/tasklin_linux_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end

    on_intel do
      url "https://github.com/yamidaisuke/tasklin/releases/download/v0.0.0/tasklin_linux_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "tasklin"
  end

  test do
    system "#{bin}/tasklin", "--version"
  end
end

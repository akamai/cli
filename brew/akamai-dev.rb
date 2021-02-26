class AkamaiDev < Formula
  desc "CLI toolkit for working with Akamai's APIs"
  homepage "https://github.com/akamai/cli"
  url "file://#{`pwd`.strip}/cli.tar.gz",
      using: :curl
  version "1.2.0"
  sha256 "9ed21a3518d2cd4b26bb74f114fb7d6d3e590139501c5b47a6bc6064de2cba32"

  bottle do
    cellar :any_skip_relocation
    sha256 "65d232aa3ff2f8a3af7a8f3d74c80539ce493bc45e2e0763e5509ad3e3ba399b" => :catalina
    sha256 "e9850383e5d94f1e8c5c1b813cdb735a81aa0e5d073e23d797cebc6e895fe96d" => :mojave
    sha256 "cf2b12e909cf2b622e2c2913dc5b20d86a45a6fa3f5b7de551dd30cde142d3df" => :high_sierra
    sha256 "a79cf0cbf309d832059bfa4a62be000ac39eb12220491ad3d93e7ca7086fcf4c" => :sierra
  end

  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    print buildpath
    srcpath = buildpath / "src/github.com/akamai/cli"
    srcpath.install buildpath.children

    cd srcpath do
      system "go", "build", "-tags", "noautoupgrade nofirstrun", "-o", bin / "akamai-dev", "cli/main.go"
      prefix.install_metafiles
    end
  end

  test do
    assert_match "Purge", shell_output("#{bin}/akamai install --force purge")
  end
end
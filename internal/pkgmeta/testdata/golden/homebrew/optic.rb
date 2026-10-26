# Note: the app is not signed or notarized, so macOS Gatekeeper blocks the
# first launch. See PACKAGING.md for the workaround.
cask "optic" do
  version "1.2.3"
  sha256 "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

  url "https://github.com/andredeomondes/optic/releases/download/v#{version}/OPTIC-macos-universal.dmg"
  name "OPTIC"
  desc "Control your TV from your PC"
  homepage "https://github.com/andredeomondes/optic"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: ">= :big_sur"

  app "OPTIC.app"
end

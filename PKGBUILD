# Maintainer: Dominik Bühler <dom.buehler@outlook.com>
pkgname=switch-tube-downloader
pkgver=0.1.0
pkgrel=1
pkgdesc="A lightweight CLI to download SwitchTube videos"
arch=('x86_64' 'aarch64')
url="https://github.com/domi413/SwitchTube-Downloader"
license=('MIT')
source_x86_64=("$pkgname-v$pkgver-linux-amd64.zip::https://github.com/domi413/SwitchTube-Downloader/releases/download/v$pkgver/switch-tube-downloader_linux_amd64.zip")
source_aarch64=("$pkgname-v$pkgver-linux-arm64.zip::https://github.com/domi413/SwitchTube-Downloader/releases/download/v$pkgver/switch-tube-downloader_linux_arm64.zip")
sha256sums_x86_64=('e0a1f0c1842f3e9f05f5c9061c62584b9040c05d89bef393f3c70585bb5100b4')
sha256sums_aarch64=('4065d8511441b2260422a6ce067188fa1661fdad4854569e438d65c1cdb9c6fc')

package() {
    # Install the binary, license, and documentation
    install -Dm755 "switch-tube-downloader" -t "$pkgdir/usr/bin"
    install -Dm644 "LICENSE" -t "$pkgdir/usr/share/licenses/$pkgname"
    install -Dm644 "README.md" -t "$pkgdir/usr/share/doc/$pkgname"
}

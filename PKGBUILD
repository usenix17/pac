# Maintainer: Sasha <sasha@starnix.net>
pkgname=pac
pkgver=0.1.0
pkgrel=4
pkgdesc='One front door for pacman (official + aur-mirror) and flatpak'
arch=('x86_64')
url='https://github.com/usenix17/pac'
license=('MIT')
depends=('pacman' 'flatpak')
# 'pac mirror' resolves AUR closures via `aur depends`; it uses local aurutils
# if present, else falls back to running it inside the builder image via docker.
# Either one satisfies 'pac mirror'; the rest of pac needs neither.
optdepends=('aurutils: AUR closure resolution for "pac mirror"'
            'docker: AUR closure resolution for "pac mirror" (fallback when aurutils is absent)')
makedepends=('go' 'git')
source=("git+https://github.com/usenix17/pac.git")
sha256sums=('SKIP')

build() {
	cd "$srcdir/pac"
	export CGO_ENABLED=0 GOFLAGS='-trimpath -mod=readonly'
	go build -o pac .
}

check() {
	cd "$srcdir/pac"
	go test ./...
}

package() {
	cd "$srcdir/pac"
	install -Dm755 pac "$pkgdir/usr/bin/pac"
}

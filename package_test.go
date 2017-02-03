package debianupdate

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/dedis/onet.v1/log"

	"testing"
)

func TestNewPackage(t *testing.T) {
	require := require.New(t)

	packageString := `Package: vim
					  Version: 2:7.4.488-7+deb8u1
					  Installed-Size: 2233
					  Maintainer: Debian Vim Maintainers <pkg-vim-maintainers@lists.alioth.debian.org>
					  Architecture: amd64
					  Provides: editor
					  Depends: vim-common (= 2:7.4.488-7+deb8u1), vim-runtime (= 2:7.4.488-7+deb8u1), libacl1 (>= 2.2.51-8), libc6 (>= 2.15), libgpm2 (>= 1.20.4), libselinux1 (>= 1.32), libtinfo5
					  Suggests: ctags, vim-doc, vim-scripts
					  Description: Vi IMproved - enhanced vi editor
					  Homepage: http://www.vim.org/
					  Description-md5: 59e8b8f7757db8b53566d5d119872de8
					  Section: editors
					  Priority: optional
					  Filename: pool/updates/main/v/vim/vim_7.4.488-7+deb8u1_amd64.deb
					  Size: 952724
					  MD5sum: 8717d2b54e532414464f0b1bde47fa51
					  SHA1: 14b243c5c9ca956c3aeaa09ad6e8debb00375a8e
					  SHA256: 537abb2e1c500aa9fd94149c9c9aeb777276b1946b13f54d1caead78e7e41a11`

	p, err := NewPackage(packageString)

	log.ErrFatal(err)
	require.NotNil(p)
}

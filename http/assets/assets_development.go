//go:build !production

package assets

import "embed"

//go:embed *.svg
//go:embed scripts/*.js
//go:embed css/theme.css
var fsys embed.FS

var IndexHTML []byte

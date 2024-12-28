//go:build production

package assets

import "embed"

//go:embed css/theme.css
var fsys embed.FS

var IndexHTML []byte

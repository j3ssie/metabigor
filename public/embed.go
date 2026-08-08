// Package public contains embedded database files for offline ASN and country lookups.
package public

import "embed"

// ASNDB contains the embedded IP-to-ASN database (zip compressed CSV).
//
//go:embed ip-to-asn.csv.zip
var ASNDB []byte

// CountryDB contains the embedded IP-to-country database (zip compressed CSV).
//
//go:embed ip-to-country.csv.zip
var CountryDB []byte

// FS provides access to all embedded files.
//
//go:embed *.zip
var FS embed.FS

// SkillsFS holds the coding-agent skill bundles shipped with the binary, under
// the skills/ directory. Exposed through the `metabigor skills` command so the
// bundled instructions always match the installed version.
//
//go:embed all:skills
var SkillsFS embed.FS

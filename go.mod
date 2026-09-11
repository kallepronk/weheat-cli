module github.com/kallepronk/weheat-cli

go 1.26.0

require (
	github.com/kallepronk/weheat-go v0.1.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/term v0.46.0
)

require golang.org/x/sys v0.48.0 // indirect

// ponytail: local sibling until weheat-go v0.1.0 is tagged and pushed; then drop this line.
replace github.com/kallepronk/weheat-go => ../weheat-go

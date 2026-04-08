package ts

import "fmt"

// Permissions controls what chart code is allowed to do.
// Follows the Deno model: deny by default, allow explicitly.
type Permissions struct {
	File    bool // --allow-file: $file.read, $file.exists
	Http    bool // --allow-http: $http.get, $http.post, etc.
	Cluster bool // --allow-cluster: $cluster.*
}

// AllPermissions returns permissions with everything enabled.
func AllPermissions() Permissions {
	return Permissions{File: true, Http: true, Cluster: true}
}

// NoPermissions returns permissions with everything denied (default).
func NoPermissions() Permissions {
	return Permissions{}
}

func denyError(global, flag string) error {
	return fmt.Errorf("%s requires %s permission. Run with --%s to allow", global, flag, flag)
}

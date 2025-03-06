// Package versions provides a hierarchical tree structure to store semantic version numbers.
// The versions are stored in an N-ary tree (major → minor → patch) so that version-related queries,
// such as finding the latest compatible version, can be performed quickly and efficiently.
package versions

import (
	"fmt"
	"sort"
)

// V represents a node in an N-ary tree for version components (major, minor, patch).
// The tree structure lets you store versions in a sorted, hierarchical manner (major → minor → patch)
// so that version-related queries (like finding the latest compatible version) are fast and straightforward.
type V struct {
	// Value represents the version number at this node (e.g. major, minor, or patch version).
	Value uint16

	// M maps a child node's value to its index in Sub.
	M map[uint16]int

	// Sub contains the child nodes which are sorted by Value.
	Sub []V
}

// leaf returns true if the node has no children.
func (v *V) leaf() bool {
	return len(v.Sub) == 0
}

// Insert adds a version path represented by a slice of uint16 values (e.g. [major, minor, patch]) into the tree.
// It recursively traverses or creates nodes corresponding to each value.
func (v *V) Insert(values []uint16) {
	if len(values) == 0 {
		return
	}
	value := values[0]

	i := sort.Search(len(v.Sub), func(i int) bool {
		return v.Sub[i].Value >= value
	})
	if i >= len(v.Sub) || v.Sub[i].Value != value {
		n := V{
			Value: value,
			M:     make(map[uint16]int),
		}
		// Insert at i
		v.Sub = append(v.Sub[:i], append(([]V{n}), v.Sub[i:]...)...)
		for j, sub := range v.Sub[i:] {
			v.M[sub.Value] = i + j
		}

	}

	v.Sub[i].Insert(values[1:])
}

func (v *V) Remove(values []uint16) {
	if len(values) == 0 {
		return
	}
	value := values[0]

	i, ex := v.M[value]
	if !ex {
		return
	}

	v.Sub[i].Remove(values[1:])

	if v.Sub[i].leaf() {
		v.Sub = append(v.Sub[:i], v.Sub[i+1:]...)
		delete(v.M, value)
		for j, sub := range v.Sub[i:] {
			v.M[sub.Value] = i + j
		}
	}
}

// Export traverses the tree and exports all stored version numbers as strings.
func (v *V) Export() (values []string) {
	values = []string{}
	for _, sub := range v.Sub {
		export := sub.Export()
		if len(export) == 0 {
			values = append(values, fmt.Sprintf("%d", sub.Value))
			continue
		}
		for _, subExp := range export {
			values = append(values, fmt.Sprintf("%d.%s", sub.Value, subExp))
		}
	}

	return
}

// Versions holds the root of the version tree. It caches the hierarchical structure
// of versions for quick lookups and manipulations.
type Versions struct {
	Tree *V
}

// New creates and returns a new Versions instance with an initialized root node.
// The root node's Value is set to 0 and is not used to represent a real version.
func New() *Versions {
	return &Versions{
		Tree: &V{
			Value: 0, // not used (root)
			M:     make(map[uint16]int),
		},
	}
}

// Add stores a version string (e.g., "1.2.3") into the version tree.
// It parses the version and inserts the major, minor, and patch values into the tree.
// If a suffix is present, the function currently does nothing.
func (sv *Versions) Add(version string) error {
	major, minor, patch, suffix, err := Parse(version)
	if err != nil {
		return err
	}
	if suffix != "" {
		// Not implemented
		return nil
	}

	sv.Tree.Insert([]uint16{major, minor, patch})

	return nil
}

// Remove deletes a version string (e.g., "1.2.3") from the version tree.
// It parses the version and removes the corresponding node path from the tree.
// Versions with a suffix are not implemented.
func (sv *Versions) Remove(version string) error {
	major, minor, patch, suffix, err := Parse(version)
	if err != nil {
		return err
	}
	if suffix != "" {
		// Not implemented
		return nil
	}

	sv.Tree.Remove([]uint16{major, minor, patch})

	return nil
}

// Export all versions. Only for testing
func (sv *Versions) Export() []string {
	return sv.Tree.Export()
}

// Empty returns true if there are no version entries stored in the tree.
func (sv *Versions) Empty() bool {
	return len(sv.Tree.Sub) == 0
}

// LatestCompatible returns the latest version that would not trigger a major upgrade,
// based on the provided current version. It finds the most recent minor and patch levels
// within the same major version (or special handling if the major version is 0).
func (sv *Versions) LatestCompatible(currentVersion string) (latestVersion string, err error) {
	latestVersion = currentVersion

	major, minor, patch, _, err := Parse(currentVersion)
	if err != nil {
		return
	}

	var latestMinor uint16
	var latestPatch uint16
	if major == 0 {
		if len(sv.Tree.Sub) == 0 || sv.Tree.Sub[0].Value != 0 {
			return
		}
		minorIndex, ex := sv.Tree.Sub[0].M[minor]
		if !ex {
			return
		}
		minorEntry := sv.Tree.Sub[0].Sub[minorIndex]

		latestMinor = minor
		if len(minorEntry.Sub) == 0 {
			panic("no patch entries under a minor entry")
		}
		latestPatch = minorEntry.Sub[len(minorEntry.Sub)-1].Value
	} else {
		majorIndex, ex := sv.Tree.M[major]
		if !ex {
			return
		}
		majorEntry := sv.Tree.Sub[majorIndex]

		if len(majorEntry.Sub) == 0 {
			panic("no minor entries under a major entry")
		}
		minorEntry := majorEntry.Sub[len(majorEntry.Sub)-1]

		latestMinor = minorEntry.Value
		if len(minorEntry.Sub) == 0 {
			panic("no patch entries under a minor entry")
		}
		latestPatch = minorEntry.Sub[len(minorEntry.Sub)-1].Value
	}

	if latestMinor < minor || latestPatch < patch {
		return
	}

	latestVersion = fmt.Sprintf("%d.%d.%d", major, latestMinor, latestPatch)
	return
}

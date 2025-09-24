package manifestmock

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/depsolvednf"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

func ResolveContainers(containerSources map[string][]container.SourceSpec) map[string][]container.Spec {
	containerSpecs := make(map[string][]container.Spec, len(containerSources))
	for plName, sourceSpecs := range containerSources {
		specs := make([]container.Spec, len(sourceSpecs))
		for idx, src := range sourceSpecs {
			digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"digest")))
			id := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"imageid")))
			listDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"list-digest")))
			name := src.Name
			if name == "" {
				name = src.Source
			}
			spec := container.Spec{
				Source:     src.Source,
				Digest:     digest,
				TLSVerify:  src.TLSVerify,
				ImageID:    id,
				LocalName:  name,
				ListDigest: listDigest,
			}
			specs[idx] = spec
		}
		containerSpecs[plName] = specs
	}
	return containerSpecs
}

func ResolveCommits(commitSources map[string][]ostree.SourceSpec) map[string][]ostree.CommitSpec {
	commits := make(map[string][]ostree.CommitSpec, len(commitSources))
	for name, commitSources := range commitSources {
		commitSpecs := make([]ostree.CommitSpec, len(commitSources))
		for idx, commitSource := range commitSources {
			commitSpecs[idx] = mockOSTreeResolve(commitSource)
		}
		commits[name] = commitSpecs
	}
	return commits
}

func Depsolve(packageSets map[string][]rpmmd.PackageSet, repos []rpmmd.RepoConfig, archName string) map[string]depsolvednf.DepsolveResult {
	depsolvedSets := make(map[string]depsolvednf.DepsolveResult)

	for name, pkgSetChain := range packageSets {
		specSet := make(rpmmd.PackageList, 0)
		seenChksumsInc := make(map[string]bool)
		seenChksumsExc := make(map[string]bool)
		for idx, pkgSet := range pkgSetChain {
			include := pkgSet.Include
			slices.Sort(include)
			for _, pkgName := range include {
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(pkgName)))
				// generate predictable but non-empty
				// release/version numbers
				ver := strconv.Itoa(int(pkgName[0]) % 9)
				rel := strconv.Itoa(int(pkgName[1]) % 9)
				spec := rpmmd.Package{
					Name:            pkgName,
					Epoch:           0,
					Version:         ver,
					Release:         rel + ".fk1",
					Arch:            archName,
					RemoteLocations: []string{fmt.Sprintf("https://example.com/repo/packages/%s", pkgName)},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: checksum},
				}
				if seenChksumsInc[spec.Checksum.String()] {
					continue
				}
				seenChksumsInc[spec.Checksum.String()] = true

				specSet = append(specSet, spec)
			}

			exclude := pkgSet.Exclude
			slices.Sort(exclude)
			for _, excludeName := range exclude {
				pkgName := fmt.Sprintf("exclude:%s", excludeName)
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(pkgName)))
				spec := rpmmd.Package{
					Name:            pkgName,
					Epoch:           0,
					Version:         "0",
					Release:         "0",
					Arch:            "noarch",
					RemoteLocations: []string{fmt.Sprintf("https://example.com/repo/packages/%s", pkgName)},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: checksum},
				}
				if seenChksumsExc[spec.Checksum.String()] {
					continue
				}
				seenChksumsExc[spec.Checksum.String()] = true

				specSet = append(specSet, spec)
			}

			// generate pseudo packages for the config of each transaction
			var setRepoNames []string
			for _, setRepo := range pkgSet.Repositories {
				setRepoNames = append(setRepoNames, setRepo.Name)
			}
			configPackageName := fmt.Sprintf("%s:transaction-%d-repos:%s", name, idx, strings.Join(setRepoNames, "+"))
			if pkgSet.InstallWeakDeps {
				configPackageName += "-weak"
			}
			depsolveConfigPackage := rpmmd.Package{
				Name:            configPackageName,
				Epoch:           0,
				Version:         "",
				Release:         "",
				Arch:            "noarch",
				RemoteLocations: []string{fmt.Sprintf("https://example.com/repo/packages/%s", configPackageName)},
				Checksum:        rpmmd.Checksum{Type: "sha256", Value: fmt.Sprintf("%x", sha256.Sum256([]byte(configPackageName)))},
				Secrets:         "",
				CheckGPG:        false,
				IgnoreSSL:       false,
				Location:        "",
				RepoID:          "",
			}
			specSet = append(specSet, depsolveConfigPackage)
		}

		// generate pseudo packages for the repos
		for _, repo := range repos {
			// the test repos have the form:
			//   https://rpmrepo..../el9/cs9-x86_64-rt-20240915
			// drop the date as it's not needed for this level of
			// mocks
			baseURL := repo.BaseURLs[0]
			if idx := strings.LastIndex(baseURL, "-"); idx > 0 {
				baseURL = baseURL[:idx]
			}
			url, err := url.Parse(baseURL)
			if err != nil {
				panic(err)
			}
			url.Host = "example.com"
			url.Path = fmt.Sprintf("passed-arch:%s/passed-repo:%s", archName, url.Path)
			specSet = append(specSet, rpmmd.Package{
				Name:            url.String(),
				RemoteLocations: []string{url.String()},
				Checksum:        rpmmd.Checksum{Type: "sha256", Value: fmt.Sprintf("%x", sha256.Sum256([]byte(url.String())))},
			})
		}

		depsolvedSets[name] = depsolvednf.DepsolveResult{
			Packages: specSet,
			Repos:    repos,
		}
	}

	return depsolvedSets
}

var OSTreeResolve = mockOSTreeResolve

func mockOSTreeResolve(commitSource ostree.SourceSpec) ostree.CommitSpec {
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(commitSource.URL+commitSource.Ref)))
	spec := ostree.CommitSpec{
		Ref:      commitSource.Ref,
		URL:      commitSource.URL,
		Checksum: checksum,
	}
	if commitSource.RHSM {
		spec.Secrets = "org.osbuild.rhsm.consumer"
	}
	return spec
}

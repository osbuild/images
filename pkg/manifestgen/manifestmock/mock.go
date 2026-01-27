package manifestmock

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
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

func Depsolve(
	packageSets map[string][]rpmmd.PackageSet,
	archName string,
	useRootDir bool,
) (map[string]depsolvednf.DepsolveResult, error) {
	depsolvedSets := make(map[string]depsolvednf.DepsolveResult)

	for pkgSetName, pkgSetChain := range packageSets {
		transactions := make(depsolvednf.TransactionList, 0, len(pkgSetChain))
		reposByHash := make(map[string]rpmmd.RepoConfig)

		// Each PackageSet in the chain represents a single transaction.
		for txIdx, pkgSet := range pkgSetChain {

			if useRootDir {
				if len(pkgSet.Repositories) > 0 {
					return nil, fmt.Errorf("package set %s has repositories when useRootDir is true", pkgSetName)
				}
				// We don't have any way to properly leverage the root dir for this mock, so we just
				// add a dummy repository to the package set.
				pkgSet.Repositories = []rpmmd.RepoConfig{
					{
						Name:     "root-dir-repo",
						BaseURLs: []string{"https://example.com/root_dir_repo"},
					},
				}
			}

			transactionPackages := make(
				rpmmd.PackageList, 0,
				len(pkgSet.Include)+len(pkgSet.Exclude)+len(pkgSet.Repositories)+1,
			)

			for _, pkgName := range pkgSet.Include {
				// Generate a unique package checksum, so that the same included package name from different
				// transactions are not considered the same package. This allows us to catch changes in the default
				// package sets when generating test manifests.
				checksum := fmt.Sprintf(
					"%x",
					sha256.Sum256([]byte(fmt.Sprintf("pkgset:%s_trans:%d_include:%s", pkgSetName, txIdx, pkgName))),
				)
				pkg := rpmmd.Package{
					// NOTE: for included packages, we use the plain package name, because some pipeline generators
					// are searching the depsolved package set for specific package names (such as 'kernel')
					// and fail if they are not found.
					Name: pkgName,
					// generate predictable but non-empty release/version numbers
					// NOTE: we can't use version higher than 4, because the OS pipeline's
					// GenDNF4VersionlockStageOptions() searches for packages with version "4"
					// to identify DNF4-related packages.
					Version:  strconv.Itoa(int(checksum[0]) % 5),
					Release:  fmt.Sprintf("%d.pkgset~%s^trans~%d", int(checksum[1])%9, pkgSetName, txIdx),
					Arch:     archName,
					Checksum: rpmmd.Checksum{Type: "sha256", Value: checksum},
				}
				pkg.RemoteLocations = []string{
					fmt.Sprintf("https://example.com/repo/packages/%s.rpm", pkg.FullNEVRA()),
				}
				transactionPackages = append(transactionPackages, pkg)
			}

			for _, pkgName := range pkgSet.Exclude {
				// Generate a unique package checksum, so that the same included package name from different
				// transactions are not considered the same package. This allows us to catch changes in the default
				// package sets when generating test manifests.
				checksum := fmt.Sprintf(
					"%x",
					sha256.Sum256([]byte(fmt.Sprintf("pkgset:%s_trans:%d_exclude:%s", pkgSetName, txIdx, pkgName))),
				)
				pkg := rpmmd.Package{
					Name: fmt.Sprintf("exclude:%s", pkgName),
					// generate predictable but non-empty release/version numbers
					Version:  strconv.Itoa(int(checksum[0]) % 9),
					Release:  fmt.Sprintf("%d.pkgset~%s^trans~%d", int(checksum[1])%9, pkgSetName, txIdx),
					Arch:     archName,
					Checksum: rpmmd.Checksum{Type: "sha256", Value: checksum},
				}
				pkg.RemoteLocations = []string{
					fmt.Sprintf("https://example.com/repo/packages/%s.rpm", pkg.FullNEVRA()),
				}
				transactionPackages = append(transactionPackages, pkg)
			}

			// generate pseudo packages for the config of each transaction
			var setRepoNames []string
			for _, setRepo := range pkgSet.Repositories {
				setRepoNames = append(setRepoNames, setRepo.Name)
			}
			configPackageName := fmt.Sprintf("pkgset:%s_trans:%d_repos:%s", pkgSetName, txIdx, strings.Join(setRepoNames, "+"))
			if pkgSet.InstallWeakDeps {
				configPackageName += "+weak"
			}
			configPkgChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(configPackageName)))
			depsolveConfigPackage := rpmmd.Package{
				Name: configPackageName,
				// generate predictable but non-empty release/version numbers
				Version:  strconv.Itoa(int(configPkgChecksum[0]) % 9),
				Release:  strconv.Itoa(int(configPkgChecksum[1])%9) + ".fk1",
				Arch:     archName,
				Checksum: rpmmd.Checksum{Type: "sha256", Value: configPkgChecksum},
			}
			depsolveConfigPackage.RemoteLocations = []string{
				fmt.Sprintf("https://example.com/repo/packages/%s.rpm", depsolveConfigPackage.FullNEVRA()),
			}
			transactionPackages = append(transactionPackages, depsolveConfigPackage)

			// Add repo pseudo-packages only for repos not seen before
			for _, repo := range pkgSet.Repositories {
				repoHash := repo.Hash()
				if _, ok := reposByHash[repoHash]; ok {
					continue
				}
				reposByHash[repoHash] = repo

				// the test repos have the form:
				//   https://rpmrepo..../el9/cs9-x86_64-rt-20240915
				// drop the date as it's not needed for this level of mocks
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
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(url.String())))
				transactionPackages = append(transactionPackages, rpmmd.Package{
					Name: url.String(),
					// generate predictable but non-empty release/version numbers
					Version:         strconv.Itoa(int(checksum[0]) % 9),
					Release:         strconv.Itoa(int(checksum[1])%9) + ".fk1",
					Arch:            archName,
					RemoteLocations: []string{url.String()},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: checksum},
				})
			}

			// The packages list in each transaction result is sorted by full NEVRA, as a real depsolver would do.
			sort.Slice(transactionPackages, func(i, j int) bool {
				return transactionPackages[i].FullNEVRA() < transactionPackages[j].FullNEVRA()
			})
			transactions = append(transactions, transactionPackages)
		}

		// Sort the list of repos by ID, as a real depsolver would do. Note that the IDs are set by the real depsolver
		// when depsolving o the Hash() method value of the RepoConfig. Therefore we sort by that value.
		allRepos := make([]rpmmd.RepoConfig, 0, len(reposByHash))
		for _, repo := range reposByHash {
			allRepos = append(allRepos, repo)
		}
		sort.Slice(allRepos, func(i, j int) bool {
			return allRepos[i].Hash() < allRepos[j].Hash()
		})

		depsolvedSets[pkgSetName] = depsolvednf.DepsolveResult{
			Packages:     transactions.AllPackages(),
			Transactions: transactions,
			Repos:        allRepos,
		}
	}

	return depsolvedSets, nil
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

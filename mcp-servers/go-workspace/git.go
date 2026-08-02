package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type ChangedFile struct {
	Absolute string
	Relative string
}

func discoverChangedFiles(ctx context.Context, resolvedRoot string, baseRef string, includeUntracked bool) ([]ChangedFile, error) {
	repoRoot, err := gitRepoRoot(ctx, resolvedRoot)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0)
	if strings.TrimSpace(baseRef) != "" {
		diffPaths, err := runLines(ctx, repoRoot, "git", "diff", "--name-only", "--diff-filter=ACMR", baseRef+"...HEAD")
		if err != nil {
			return nil, fmt.Errorf("read git diff for %s: %w", baseRef, err)
		}
		paths = append(paths, diffPaths...)
	} else {
		unstaged, err := runLines(ctx, repoRoot, "git", "diff", "--name-only", "--diff-filter=ACMR")
		if err != nil {
			return nil, fmt.Errorf("read unstaged git diff: %w", err)
		}
		staged, err := runLines(ctx, repoRoot, "git", "diff", "--cached", "--name-only", "--diff-filter=ACMR")
		if err != nil {
			return nil, fmt.Errorf("read staged git diff: %w", err)
		}
		paths = append(paths, unstaged...)
		paths = append(paths, staged...)
		if includeUntracked {
			untracked, err := runLines(ctx, repoRoot, "git", "ls-files", "--others", "--exclude-standard")
			if err != nil {
				return nil, fmt.Errorf("read untracked files: %w", err)
			}
			paths = append(paths, untracked...)
		}
	}

	paths = uniqueStrings(paths)
	files := make([]ChangedFile, 0, len(paths))
	for _, path := range paths {
		absolute := filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if !isWithinRoot(resolvedRoot, absolute) {
			continue
		}
		relativeToRoot, err := filepath.Rel(resolvedRoot, absolute)
		if err != nil {
			return nil, fmt.Errorf("relativize %s to %s: %w", absolute, resolvedRoot, err)
		}
		files = append(files, ChangedFile{
			Absolute: absolute,
			Relative: filepath.ToSlash(relativeToRoot),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Relative < files[j].Relative
	})

	return files, nil
}

func gitRepoRoot(ctx context.Context, resolvedRoot string) (string, error) {
	output, err := runCommand(ctx, resolvedRoot, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve git repo root from %s: %w", resolvedRoot, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func runLines(ctx context.Context, dir string, name string, args ...string) ([]string, error) {
	output, err := runCommand(ctx, dir, name, args...)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		values = append(values, line)
	}
	return values, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func changedPackages(files []ChangedFile, packages []PackageInfo, resolvedRoot string) ([]PackageSummary, []string) {
	byDir := make(map[string]PackageInfo, len(packages))
	for _, pkg := range packages {
		byDir[filepath.Clean(pkg.Dir)] = pkg
	}

	matched := make(map[string]PackageSummary, len(packages))
	unmatched := make([]string, 0)
	for _, file := range files {
		pkg, ok := nearestPackageForFile(file.Absolute, byDir, resolvedRoot)
		if !ok {
			unmatched = append(unmatched, file.Relative)
			continue
		}
		matched[pkg.ImportPath] = pkg.Summary()
	}

	summaries := make([]PackageSummary, 0, len(matched))
	for _, summary := range matched {
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ImportPath < summaries[j].ImportPath
	})
	sort.Strings(unmatched)

	return summaries, unmatched
}

func nearestPackageForFile(file string, packagesByDir map[string]PackageInfo, stopRoot string) (PackageInfo, bool) {
	current := filepath.Dir(file)
	stopRoot = filepath.Clean(stopRoot)

	for {
		if pkg, ok := packagesByDir[current]; ok {
			return pkg, true
		}
		if current == stopRoot {
			break
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	return PackageInfo{}, false
}

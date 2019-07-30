package main

import (
	"gopkg.in/src-d/go-git.v4"
)

// cloneSytest is a function that clones the Sytest source code
func cloneSytest(dir, url string) (err error) {
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL:   url,
		Depth: 1,
	})
	return
}

// pullSytest is a function that pulls down the latest changes in an existing
// Sytest checkout
func pullSytest(dir string) (err error) {
	// Open the repository
	var r *git.Repository
	r, err = git.PlainOpen(dir)
	if err != nil {
		return
	}

	// Get the working directory for the repository
	var w *git.Worktree
	w, err = r.Worktree()
	if err != nil {
		return
	}

	// Pull the latest changes from the origin remote and merge into the current branch
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
	})
	// Ignore error if it reports that it's already up to date
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return
}

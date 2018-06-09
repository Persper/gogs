// Copyright 2018 The Persper Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// The user table in the IPFS
type DeUser struct {
	ID                 int64
	Name               string `xorm:"UNIQUE NOT NULL"`
	FullName           string
	Email              string `xorm:"NOT NULL"`
	Passwd             string `xorm:"NOT NULL"`
	LoginType          LoginType
	LoginSource        int64 `xorm:"NOT NULL DEFAULT 0"`
	LoginName          string
	Location           string
	Website            string
	Rands              string `xorm:"VARCHAR(10)"`
	Salt               string `xorm:"VARCHAR(10)"`
	CreatedUnix        int64
	LastRepoVisibility bool
	Avatar             string `xorm:"VARCHAR(2048) NOT NULL"`
	AvatarEmail        string `xorm:"NOT NULL"`
	UseCustomAvatar    bool
	//user.email_address.email[]
	//star.repo_id[]
	//watch.repo.id[]
	//repo_blacklist
	//team_blacklist
	//org_blacklist
}

func transferUserToDeUser(deUser *DeUser, user *User) {
	deUser.ID = user.ID
	deUser.Name = user.Name
	deUser.FullName = user.FullName
	deUser.Email = user.Email
	deUser.Passwd = user.Passwd
	deUser.LoginType = user.LoginType
	deUser.LoginSource = user.LoginSource
	deUser.LoginName = user.LoginName
	deUser.Location = user.Location
	deUser.Website = user.Website
	deUser.Rands = user.Rands
	deUser.Salt = user.Salt
	deUser.CreatedUnix = user.CreatedUnix
	deUser.LastRepoVisibility = user.LastRepoVisibility
	deUser.Avatar = user.Avatar
	deUser.AvatarEmail = user.AvatarEmail
	deUser.UseCustomAvatar = user.UseCustomAvatar
}

func deTransferUserToDeUser(deUser *DeUser, user *User) error {
	user.ID = deUser.ID
	user.Name = deUser.Name
	user.FullName = deUser.FullName
	user.Email = deUser.Email
	user.Passwd = deUser.Passwd
	user.LoginType = deUser.LoginType
	user.LoginSource = deUser.LoginSource
	user.LoginName = deUser.LoginName
	user.Location = deUser.Location
	user.Website = deUser.Website
	user.Rands = deUser.Rands
	user.Salt = deUser.Salt
	user.CreatedUnix = deUser.CreatedUnix
	user.LastRepoVisibility = deUser.LastRepoVisibility
	user.Avatar = deUser.Avatar
	user.AvatarEmail = deUser.AvatarEmail
	user.UseCustomAvatar = deUser.UseCustomAvatar

	// recovery deUser to user
	user.Type = USER_TYPE_INDIVIDUAL
	user.LowerName = strings.ToLower(user.Name)
	user.UpdatedUnix = user.CreatedUnix
	user.MaxRepoCreation = -1
	user.IsAdmin = false
	user.Description = ""
	user.NumTeams = 0
	user.NumMembers = 0

	// TODO: the follow and star table is lost
	var err error
	var total int64

	follow := new(Follow)
	total, err = x.Where("follow_id = ?", user.ID).Count(follow)
	if err != nil {
		return fmt.Errorf("Can not get user numfollowers: %v", err)
	}
	user.NumFollowers = int(total)

	total, err = x.Where("user_id = ?", user.ID).Count(follow)
	if err != nil {
		return fmt.Errorf("Can not get user numfollowing: %v", err)
	}
	user.NumFollowing = int(total)

	// star is useless
	user.NumStars = 0

	repo := new(Repository)
	total, err = x.Where("owner_id = ?", user.ID).Count(repo)
	if err != nil {
		return fmt.Errorf("Can not get user numRepos: %v", err)
	}
	user.NumRepos = int(total)

	// TODO: not sure
	// user.UportId
	user.IsActive = true
	user.AllowGitHook = false
	user.AllowImportLocal = false
	user.ProhibitLogin = false

	return nil
}

/// Push the user info to IPFS and record the new ipfsHash in the blockchain
/// pushMode: 0 - register; 1 - update; 2 - delete;
func PushUserInfo(contextUser *User， pushMode int) (err error) {
	// Do some checks
	if contextUser.IsOrganization() {
		return nil
	}
	if !canPushToBlockchain(contextUser) {
		return fmt.Errorf("The user can not push to the blockchain")
	}

	// Get the corresponding user.
	var user *User
	user = &User{ID: contextUser.ID}
	hasUser, err := x.Get(user)
	if err != nil {
		return fmt.Errorf("Can not get user data: %v", err)
	}

	if hasUser {
		// Step1: register/deregister the user if it does not exist
		if pushMode == 1 {
			//registerName
		}
		if pushMode == 2 {
			//deregisterName
			return nil
		}

		// Step 2: Encode user data into JSON format
		deUser := new(DeUser)
		transferUserToDeUser(deUser, user)
		user_data, err := json.Marshal(deUser)
		if err != nil {
			return fmt.Errorf("Can not encode user data: %v", err)
		}

		// Step 3: Put the encoded data into IPFS
		c := fmt.Sprintf("echo '%s' | ipfs add ", user_data)
		cmd := exec.Command("sh", "-c", c)
		out, err2 := cmd.Output()
		if err2 != nil {
			return fmt.Errorf("Push User to IPFS: fails: %v", err2)
		}
		ipfsHash := strings.Split(string(out), " ")[1]

		// Step4: Modify the ipfsHash in the smart contract
		// TODO: setUserInfo(ipfsHash)
		ipfsHash = ipfsHash
	}

	return nil
}

// Get the new ipfsHash from the blockchain and get the user info from IPFS
func GetUserInfo(contextUser *User) (err error) {
	// Step1: get the user info hash via addrToUserInfo
	ipfsHash := "QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"

	// Step2: get the ipfs file and get the user data
	c := fmt.Sprintf("ipfs cat ", ipfsHash)
	cmd := exec.Command("sh", "-c", c)
	user_data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Get User data from IPFS: fails: %v", err)
	}

	// Step3: unmarshall user data
	newDeUser := new(DeUser)
	err = json.Unmarshal(user_data, &newDeUser)
	if err != nil {
		return fmt.Errorf("Can not decode data: %v", err)
	}

	newUser := new(User)
	deTransferUserToDeUser(newDeUser, newUser)

	// Step4: write into the local database
	// TODO:
	CreateUser(newUser)

	return nil
}
package utils

import (
	log "github.com/Sirupsen/logrus"
	gov "github.com/hashicorp/go-version"
)

// CheckVersion check version
func CheckVersion(v1, v2, gitHead1, gitHead2 string) (mark bool) {
	var (
		gov1 *gov.Version
		gov2 *gov.Version
		err  error
	)

	if gov1, err = gov.NewVersion(v1); err != nil {
		log.Fatalf("Format version v1[%s] error:%v", v1, err)
	}

	if gov2, err = gov.NewVersion(v2); err != nil {
		log.Fatalf("Format version v2[%s] error:%v", v2, err)
	}

	if mark = gov1.Equal(gov2); mark {
		mark = gitHead1 == gitHead2
	}

	return
}

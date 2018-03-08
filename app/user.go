package app

import (
	"fmt"
	"github.com/manager/vendor"
	"os"
)

// EnrollUser 用户登录
func EnrollUser() {
	setup := vendor.BaseSetupImpl{
		ConfigFile: "./" + vendor.ConfigFile,
	}

	fabricConfig, err := setup.InitConfig()()
	if err != nil {
		fmt.Printf("Failed InitConfig [%s]\n", err)
		os.Exit(1)
	}
	fmt.Println(fabricConfig)
	fmt.Println("InitConfig succeed")
}


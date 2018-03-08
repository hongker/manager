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

	if err = vendor.Enroll(fabricConfig, "admin", "adminpw", "org1"); err != nil {
		fmt.Println("Enroll failed:"+err.Error())
		os.Exit(1)
	}
	fmt.Println("Login succeed")
}


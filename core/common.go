package core

import (
	"fmt"
	"io/ioutil"
	"os"

	// "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// InitLog init log
func InitLog(options Options) {
	log.SetOutput(os.Stderr)
	if options.Debug == true {
		log.SetLevel(logrus.DebugLevel)
	} else if options.Verbose == true {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

// GoodF print good message
func GoodF(format string, args ...interface{}) {
	good := color.HiGreenString("[+]")
	fmt.Printf("%s %s\n", good, fmt.Sprintf(format, args...))
}

// BannerF print info message
func BannerF(format string, data string) {
	banner := color.BlueString("[*] %v", format)
	log.Info(fmt.Sprintf("%v%v", banner, color.HiGreenString(data)))
}

// InforF print info message
func InforF(format string, args ...interface{}) {
	log.Info(fmt.Sprintf(format, args...))
}

// WarningF print good message
func WarningF(format string, args ...interface{}) {
	good := color.YellowString("[!]")
	fmt.Printf("%s %s\n", good, fmt.Sprintf(format, args...))
}

// DebugF print debug message
func DebugF(format string, args ...interface{}) {
	log.Debug(fmt.Sprintf(format, args...))
}

// ErrorF print good message
func ErrorF(format string, args ...interface{}) {
	good := color.RedString("[-]")
	fmt.Printf("%s %s\n", good, fmt.Sprintf(format, args...))
}

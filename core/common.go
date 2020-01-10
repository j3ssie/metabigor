package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	// "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// InitLog init log
func InitLog(options Options) {
	logDir := options.Scan.TmpOutput
	if logDir == "" {
		logDir = os.TempDir()
	}
	if !FolderExists(logDir) {
		os.MkdirAll(logDir, 0755)
	}
	logFile := path.Join(logDir, "metabigor.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	// defer f.Close()
	mwr := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mwr)

	if options.Debug == true {
		log.SetOutput(mwr)
		log.SetLevel(logrus.DebugLevel)
	} else if options.Verbose == true {
		log.SetOutput(mwr)
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetOutput(ioutil.Discard)
	}
	log.Info(fmt.Sprintf("Store log file to: %v", logFile))
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

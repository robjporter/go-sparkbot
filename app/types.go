package app

import (
	"github.com/Sirupsen/logrus"
	"github.com/kataras/iris"
	"github.com/robjporter/go-utils/filesystem/config"
)

type Application struct {
	conf   *config.Config
	Log    *logrus.Logger
	Server *iris.Application
}

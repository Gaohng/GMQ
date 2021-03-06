package core

import (
	"github.com/gw123/GMQ/common/common_types"
	"github.com/gw123/GMQ/core/interfaces"
	"encoding/json"
	"github.com/spf13/viper"
	"github.com/jinzhu/gorm"
)

type App struct {
	errorManager      *ErrorManager
	moduleManager     *ModuleManager
	configManager     *ConfigManager
	middlewareManager *MiddlewareManager
	logManager        *LogManager
	dispatch          *Dispatch
	appEventNames     string
	Version           string
	configData        *viper.Viper
	DbPool            interfaces.DbPool
}

func NewApp(viper2 *viper.Viper) *App {
	this := &App{}
	this.configData = viper2

	this.Version = "2.0.0"
	this.dispatch = NewDispath(this)
	this.logManager = NewLogManager(this)
	this.logManager.Start()

	this.configManager = NewConfigManager(this, viper2)
	this.moduleManager = NewModuleManager(this, this.configManager)
	this.errorManager = NewErrorManager(this)
	this.middlewareManager = NewMiddlewareManager(this)

	this.DbPool = NewDbPool()
	return this
}

func (this *App) Start() {
	this.LoadDb()
	go this.doWorker()
}

//加载数据库配置
func (this *App) LoadDb() {
	configs := this.configData.GetStringMap("dbpool")
	for key, config := range configs {
		configMap, ok := config.(map[string]interface{})
		if ok {
			drive, ok := configMap["drive"].(string)
			if !ok {
				drive = "mysql"
			}
			host, ok := configMap["host"].(string)
			if !ok {
				host = "127.0.0.1"
			}
			database, ok := configMap["database"].(string)
			if !ok {
				database = ""
			}
			username, ok := configMap["username"].(string)
			if !ok {
				database = "root"
			}
			password, ok := configMap["password"].(string)
			if !ok {
				password = ""
			}

			db, err := this.DbPool.NewDb(
				drive,
				host,
				database,
				username,
				password);
			if err != nil {
				this.Warning("App", "db load error, %s: ", err.Error())
			}
			this.DbPool.SetDb(key, db)
		}
	}

	//set default db
	defaultDBkey, ok := configs["default"].(string)
	if !ok {
		return
	}

	this.Debug("App", "default DB  key", defaultDBkey)

	db, err := this.DbPool.GetDb(defaultDBkey)
	if err != nil {
		this.Warning("App", "not found  db  default config :%s", err.Error())
	} else {
		this.DbPool.SetDb("default", db)
	}
}

func (this *App) doWorker() {
	this.Debug("App", "Load modules")
	this.moduleManager.LoadModules()
	this.appEventNames = "stopModule,startModule,configChange"
	this.dispatch.SetEventNames(this.appEventNames)
	event := common_types.NewEvent("appReady", []byte{})
	this.Pub(event)
	go this.dispatch.Start()
}

func (this *App) Handel(event interfaces.Event) {
	//this.Debug("App", "App event"+event.GetEventName())
	switch event.GetEventName() {
	case "configChange":
		mconfig := &ModuleConfig{}
		json.Unmarshal(event.GetPayload(), mconfig)
		moduleName := mconfig.GetModuleName()
		oldModuleConfig := this.configManager.ModuleConfigs[moduleName]
		newConfigs := mconfig.GetItems()
		for key, val := range newConfigs {
			oldModuleConfig.SetItem(key, val)
		}
		break
	case "stopModule":
		moduleName := string(event.GetPayload())
		this.moduleManager.UnLoadModule(moduleName)
		break
	case "startModule":
		moduleName := string(event.GetPayload())
		moduleConfig := this.configManager.ModuleConfigs[moduleName]
		if moduleConfig == nil {
			moduleConfig = NewModuleConfig(moduleName, this.configManager.GlobalConfig)
		}
		this.moduleManager.LoadModule(moduleName, moduleConfig)
		break
	}
}

func (this *App) Sub(eventName string, module interfaces.Module) {
	if this.dispatch != nil {
		this.dispatch.Sub(eventName, module)
	} else {
		this.Error("App", "dispath unready")
	}
}

func (this *App) UnSub(eventName string, module interfaces.Module) {
	if this.dispatch != nil {
		this.dispatch.UnSub(eventName, module)
	} else {
		this.Error("App", "dispath unready")
	}
}

func (this *App) Pub(event interfaces.Event) {
	if this.middlewareManager.Handel(event) {
		this.dispatch.Pub(event)
	}
}

func (this *App) Info(category string, format string, a ...interface{}) {
	if format == "" {
		return
	}
	this.logManager.Info(category, format, a...)
}

func (this *App) Warning(category string, format string, a ...interface{}) {
	if format == "" {
		return
	}
	this.logManager.Waring(category, format, a...)
}

func (this *App) Error(category string, format string, a ...interface{}) {
	if format == "" {
		return
	}
	this.logManager.Error(category, format, a...)
}

func (this *App) Debug(category string, format string, a ...interface{}) {
	if format == "" {
		return
	}
	this.logManager.Debug(category, format, a...)
}

func (this *App) GetVersion() string {
	return this.Version
}

func (this *App) GetConfigItem(section, key string) (string, error) {
	sect := this.configData.GetStringMapString(section)
	if sect != nil {
		return "", nil
	}
	val, ok := sect[key]
	if !ok {
		return "", nil
	}
	return val, nil
}

func (this *App) GetDefaultConfigItem(key string) (string, error) {
	return this.GetConfigItem("app", key)
}

/***/
func (this *App) LoadModuleProvider(provider interfaces.ModuleProvider) {
	this.moduleManager.LoadModuleProvider(provider)
}

//获取数据库信息
func (this *App) GetDb(dbname string) (*gorm.DB, error) {
	if dbname == "" {
		return this.GetDefaultDb()
	}
	return this.DbPool.GetDb(dbname)
}

//获取默认数据库
func (this *App) GetDefaultDb() (*gorm.DB, error) {
	dbname := "default"
	return this.GetDb(dbname)
}

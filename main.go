package main

import (
	"context"
	"errors"
	"main/admin"
	"main/conf"
	"main/database/pgsql"
	"main/database/redis"
	"main/judge"
	"main/submission"
	"main/system"
	"main/util/log"
	"main/util/password"
	"main/util/token"
	"os"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"github.com/wneessen/go-mail"
)

func main() {
	//初始化全局组件
	config, err := conf.ReadConfig()
	if err != nil {
		panic(err)
	}

	log.InitLoggr(config.Log)
	log.Logger.Debug("日志系统初始化成功")

	err = pgsql.Start(config.DB)
	if err != nil {
		log.Logger.Fatal(err)
	}
	log.Logger.Debug("数据库连接成功")

	//初始化redis
	err = redis.InitRedis(config.Redis)
	if err != nil {
		log.Logger.Fatal(errors.New("Redis连接失败: " + err.Error()))
		os.Exit(1)
	}
	log.Logger.Debug("Redis连接成功")

	//初始化JWT系统
	err = token.InitJWT(config.System.Token)
	if err != nil {
		log.Logger.Fatal(errors.New("JWT系统初始化失败: " + err.Error()))
		os.Exit(1)
	}
	log.Logger.Debug("JWT系统初始化成功")

	if !checkSystemInit() {
		log.Logger.System("检查到系统未完成初始化，开始初始化系统")
		err := initSystem(config.System)
		if err != nil {
			log.Logger.Fatal(errors.New("系统初始化失败: " + err.Error()))
			os.Exit(1)
		}
		log.Logger.System("系统初始化成功")
	}

	// TODO 从数据库读取内容，继续各项动作
	// 启动全局系统
	system.SysStart(config.API)

	// 启动管理后台服务
	go admin.InitRouter(config.API) //TODO 使用管道查询启动状态并进行控制

	// 启动评委后台服务
	go judge.InitRouter(config.API)

	//启动投稿后台
	submission.InitRouter(config.API)
}

func checkSystemInit() bool {
	isInit, err := pgsql.CheckIfSystemInit()
	if err != nil {
		log.Logger.Fatal(errors.New("检查系统初始化状态失败: " + err.Error()))
		os.Exit(1)
	}

	return isInit
}

func initSystem(systemConf conf.SystemConfig) error {
	//初始化邮件系统
	err := initEmailSystem(systemConf.Email)
	if err != nil {
		return err
	}

	//初始化超级管理员账号
	err = initSuperAdminAccount()
	if err != nil {
		return err
	}

	// 修改初始化位
	err = pgsql.ChangeSystemInitStatus(true)
	if err != nil {
		return err
	}

	return nil
}

func initEmailSystem(emailConf conf.EmailConfig) error {
	//配置客户端
	c, err := mail.NewClient(emailConf.SMTPHost, mail.WithPort(emailConf.SMTPPort), mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(emailConf.EmailAddress), mail.WithPassword(emailConf.EmailAPPPassword))
	if err != nil {
		return uerr.NewError(errors.New("创建邮件客户端失败: " + err.Error()))
	}

	//测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = c.DialWithContext(ctx)
	if err != nil {
		// 处理连接或认证失败
		return uerr.NewError(err)
	}
	defer c.Close()

	// 向数据库写入邮件配置
	err = pgsql.WriteSystemEmailConfig(emailConf)
	if err != nil {
		return uerr.NewError(errors.New("写入邮件配置到数据库失败: " + err.Error()))
	}

	return nil
}

func initSuperAdminAccount() error {
	admin, err := pgsql.FindAdminByUsername("superadmin")
	if err != nil {
		return uerr.NewError(errors.New("查询默认超级管理员账号失败: " + err.Error()))
	}

	//生成随机密码
	randNewPassword := password.Generate()
	//hash密码
	hashedPassword, err := password.HashPassword(randNewPassword)
	if err != nil {
		return uerr.NewError(errors.New("生成超级管理员账号密码失败: " + err.Error()))
	}
	//更新数据库中的超级管理员密码
	err = pgsql.ChangeAdminPassword(admin.AdminID, hashedPassword)
	if err != nil {
		return uerr.NewError(errors.New("更新超级管理员账号密码失败: " + err.Error()))
	}

	//打印新密码
	log.Logger.System("默认超级管理员账号密码已重置为:  " + randNewPassword + "  ,使用用户名: superadmin  登录")
	return nil
}

package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/gookit/color"
	"github.com/olekukonko/tablewriter"
	"os"
	"os/exec"
)

var (
	ErrNotGitDir    = errors.New("not git dir")
	ErrCguUserExist = errors.New("cgu user exist")
)
var GithubIssuesUrl = "www.baidu.com"

func main() {
	var err error
	arg := os.Args
	if len(arg) > 1 {
		switch arg[1] {
		case "ls":
			fallthrough
		case "list":
			err = getList()
		case "add":
			err = addUser()
		case "use":
			err = chooseUser(false)
		case "delete":
			fallthrough
		case "del":
			err = delUser()
		case "global":
			err = chooseUser(true)
		}
	} else {
		//列出使用说明
		color256 := color.C256(212)
		color256.Print("cgu ls")
		fmt.Println("  查看当前目录git仓库所使用用户，及用户列表")
		color256.Print("cgu use")
		fmt.Println(" 切换当前目录的git仓库用户")
		color256.Print("cgu add")
		fmt.Println(" 添加git用户")
		color256.Print("cgu del")
		fmt.Println(" 删除git用户")
		err = getList()
	}

	if err != nil {
		if errors.Is(err, ErrNotGitDir) {
			fmt.Print("\n  ")
			notGitNote()
			fmt.Print("\n")
		} else {
			fmt.Println("预料外的错误，请帮助我将该错误粘贴到"+GithubIssuesUrl+"：", err)
		}
	}
}

func getList() error {
	allUsers, err := getAllUser()
	if err != nil {
		return err
	}
	gName := allUsers[0][0]
	gEmail := allUsers[0][1]
	header := []string{"用户名", "邮箱"}

	if !isGitDir() { // 不是git仓库
		//notGitNote()
	} else {
		//获取当前仓库的name
		pName, pEmail, err := getProjectUser()
		if err != nil {
			return err
		}
		nowGitPath, err := getNowGitPath()
		if err != nil {
			return err
		}
		if pName == "" {
			pName = gName
		}
		if pEmail == "" {
			pEmail = gEmail
		}
		color256 := color.C256(211)
		fmt.Print("当前目录使用 name=")
		color256.Print(pName)
		fmt.Print(" email=")
		color256.Print(pEmail)
		fmt.Println(" (作用于" + nowGitPath + ")")
	}

	showTable(allUsers, header)
	return nil
}

func notGitNote() {
	color256 := color.C256(212)
	fmt.Print("当前目录不是git仓库，无法使用 ")
	color256.Print("cgu use")
	fmt.Print(" 切换用户，如要切换全局用户请使用 ")
	color256.Println("cgu global")
}

func getAllUser() ([][]string, error) {
	gName, gEmail, err := getGlobalUser()
	if err != nil {
		return nil, err
	}
	//获取用户前先尝试将全局用户写入
	if err = writeCguUser(gName, gEmail); err != nil && !errors.Is(err, ErrCguUserExist) {
		return nil, err
	}

	var allUsers [][]string
	allUsers = append(allUsers, []string{gName, gEmail})

	cguConfigPath, err := getCguConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := ini.Load(cguConfigPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}
	secs := cfg.Sections()
	for _, v := range secs[1:] {
		if v.Key("name").String() != gName || v.Key("email").String() != gEmail {
			var tmp []string
			tmp = append(tmp, v.Key("name").String())
			tmp = append(tmp, v.Key("email").String())
			allUsers = append(allUsers, tmp)
		}
	}

	return allUsers, nil
}

func getNowSelectedUser(allUser [][]string) (int, error) {
	pName, pEmail, err := getProjectUser()
	if err != nil {
		return 0, err
	}
	if pName == "" {
		pName = allUser[0][0]
	}
	if pEmail == "" {
		pEmail = allUser[0][1]
	}
	for k, v := range allUser {
		if v[0] == pName && v[1] == pEmail {
			return k, nil
		}
	}
	return -1, nil
}

func showTable(data [][]string, header []string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{tablewriter.Bold})
	oldChoice, err := getNowSelectedUser(data)
	for i, row := range data {
		if err == nil && oldChoice >= 0 && i == oldChoice {
			row[0] = "* " + row[0]
			table.Rich(row, []tablewriter.Colors{{tablewriter.Bold}, {tablewriter.Bold}})
		} else {
			row[0] = "  " + row[0]
			table.Append(row)
		}
	}
	table.Render()
}

func getGlobalUser() (string, string, error) {
	globalPath, err := getGlobalGitPath()
	if err != nil {
		return "", "", err
	}
	cfg, err := ini.Load(globalPath)
	if err != nil {
		return "", "", fmt.Errorf("加载配置文件失败: %w", err)
	}
	name := cfg.Section("user").Key("name").String()
	email := cfg.Section("user").Key("email").String()
	return name, email, nil
}

func getGlobalGitPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取git全局配置文件失败: %w", err)
	}
	return homeDir + "/.gitconfig", nil
}
func getProjectGitPath() (string, error) {
	projectGitPath, err := exec.Command("git", "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("获取git项目配置文件失败: %w", err)
	}
	return string(projectGitPath[0:len(projectGitPath)-1]) + "/config", nil
}

func isGitDir() bool {
	isGit, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	if err != nil {
		return false
	}

	if len(isGit) > 0 && string(isGit[0:len(isGit)-1]) == "true" {
		return true
	}
	return false
}

func getNowGitPath() (string, error) {
	gitPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("获取git项目配置文件失败: %w", err)
	}
	return string(gitPath[0 : len(gitPath)-1]), nil
}

func getProjectUser() (string, string, error) {
	projectPath, err := getProjectGitPath()
	if err != nil {
		return "", "", err
	}
	cfg, err := ini.Load(projectPath)
	if err != nil {
		return "", "", fmt.Errorf("加载配置文件失败: %w", err)
	}
	name := cfg.Section("user").Key("name").String()
	email := cfg.Section("user").Key("email").String()
	return name, email, nil
}

func getCguConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取cgu配置文件失败: %w", err)
	}
	return homeDir + "/.cguconfig", nil
}

func writeCguUser(name string, email string) error {
	cguConfigPath, err := getCguConfigPath()
	if err != nil {
		//创建文件
		f, err := os.Create(cguConfigPath)
		if err != nil {
			return fmt.Errorf("创建配置文件失败: %w", err)
		}
		_ = f.Close()
	}

	md5Str := fmt.Sprintf("%x", md5.Sum([]byte(name+email)))

	cfg, err := ini.Load(cguConfigPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}
	if cfg.Section(md5Str).Key("name").String() == "" {
		cfg.Section(md5Str).Key("name").SetValue(name)
		cfg.Section(md5Str).Key("email").SetValue(email)
		err = cfg.SaveTo(cguConfigPath)
		if err != nil {
			return fmt.Errorf("写入配置文件失败: %w", err)
		}
		return nil
	} else {
		return ErrCguUserExist
	}
}

func doUse(name string, email string, isGlobal bool) error {
	if isGlobal {
		_, err := exec.Command("git", "config", "--global", "user.name", name).Output()
		if err != nil {
			return fmt.Errorf("配置全局用户name失败: %w", err)
		}
		_, err = exec.Command("git", "config", "--global", "user.email", email).Output()
		if err != nil {
			return fmt.Errorf("配置全局用户email失败: %w", err)
		}
	} else {
		_, err := exec.Command("git", "config", "user.name", name).Output()
		if err != nil {
			return fmt.Errorf("配置当前用户name失败: %w", err)
		}
		_, err = exec.Command("git", "config", "user.email", email).Output()
		if err != nil {
			return fmt.Errorf("配置当前用户email失败: %w", err)
		}
	}
	return nil
}

func doDel(name string, email string) error {
	md5Str := fmt.Sprintf("%x", md5.Sum([]byte(name+email)))
	cguConfigPath, err := getCguConfigPath()
	if err != nil {
		return err
	}
	cfg, err := ini.Load(cguConfigPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}
	cfg.DeleteSection(md5Str)
	err = cfg.SaveTo(cguConfigPath)
	if err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return nil
}

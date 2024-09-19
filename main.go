package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

var zshrc = `source $HOME/.antigen.zsh
antigen use oh-my-zsh
antigen bundle zsh-users/zsh-autosuggestions
antigen bundle zsh-users/zsh-syntax-highlighting
antigen bundle git
antigen bundle brew
antigen bundle pip
antigen bundle aliases
antigen theme clean
antigen apply

# bun completions
[ -s "$HOME/.bun/_bun" ] && source "$HOME/.bun/_bun"

#bun
export BUN_INSTALL="$HOME/.bun" 
export PATH="$BUN_INSTALL/bin:$PATH"
`

func runCmd(cmd string, printOutput bool) {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println(err)
	} else {
		if printOutput && len(out) > 0 {
			str := strings.TrimSpace(string(out))
			fmt.Println(str)
		}
	}
}

func askInput(question string, reader *bufio.Reader) (string, error) {
	input := ""
	fmt.Print(question)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func askPassword(question string) (string, error) {
	fmt.Print(question)
	bytesInput, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return strings.TrimSpace(string(bytesInput)), nil
}

func main() {
	verbose := false
	for _, v := range os.Args[1:] {
		switch v {
		case "verbose":
			verbose = true
		case "-v":
			verbose = true
		case "--verbose":
			verbose = true
		}
	}
	reader := bufio.NewReader(os.Stdin)

	runCmd("apt-get install -y -qq unzip", verbose)
	runCmd("curl -fsSL https://bun.sh/install | bash", verbose)
	runCmd("apt-get update -y -qq && apt-get upgrade -y -qq", verbose)

	packages := []string{
		"sudo", "ufw", "fail2ban", "htop", "curl", "nginx", "tmux", "git",
		"certbot", "python3-certbot-dns-cloudflare", "autojump", "zsh", "rustc",
		"golang", "nmap", "ffmpeg",
	}

	for _, pkg := range packages {
		runCmd(fmt.Sprintf("apt-get -y -qq install %v", pkg), verbose)
	}

	_, err := exec.LookPath("docker")
	if err != nil {
		runCmd("curl -fsSL https://get.docker.com | bash", verbose)
	}

	runCmd("curl -L git.io/antigen > /root/.antigen.zsh", verbose)

	file, err := os.Create("/root/.zshrc")
	if err != nil {
		panic(err)
	}

	_, err = file.Write([]byte(zshrc))
	if err != nil {
		panic(err)
	}

	err = file.Close()
	if err != nil {
		panic(err)
	}

	runCmd("sed -i '/PermitRootLogin yes/c\\PermitRootLogin no' /etc/ssh/sshd_config", verbose)
	runCmd("sed -i '/PasswordAuthentication yes/c\\PasswordAuthentication no' /etc/ssh/sshd_config", verbose)
	runCmd("sed -i '/#Port 22/c\\Port 2218' /etc/ssh/sshd_config", verbose)
	runCmd("sed -i '/Port 22/c\\Port 2218' /etc/ssh/sshd_config", verbose)
	runCmd("sed -i '/Port 2222/c\\Port 2218' /etc/ssh/sshd_config", verbose)
	runCmd("rm -rf /etc/fail2ban/jail.local", verbose)
	runCmd("cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local", verbose)
	runCmd("sed -i '/backend = %(sshd_backend)s/c\\sshd_backend = systemd\\nbackend = %(sshd_backend)s' /etc/fail2ban/jail.local", verbose)
	runCmd("sed -i '/bantime  = 10m/c\\bantime  = 60m' /etc/fail2ban/jail.local", verbose)
	runCmd("sed -i '/findtime  = 10m/c\\findtime  = 60m' /etc/fail2ban/jail.local", verbose)
	runCmd("sed -i '/maxretry = 5/c\\maxretry = 3' /etc/fail2ban/jail.local", verbose)
	runCmd("systemctl restart ssh", verbose)
	runCmd("systemctl restart fail2ban", verbose)

	fmt.Println()
	fmt.Print("SSH: ", Green)
	runCmd("systemctl is-active ssh", true)
	fmt.Print(Reset)
	fmt.Print("Fail2ban: ", Green)
	runCmd("systemctl is-active fail2ban", true)
	fmt.Print(Reset)

	runCmd("chsh -s /usr/bin/zsh", verbose)
	runCmd("timedatectl set-timezone America/Phoenix", verbose)

	fmt.Println()
	username, password := "", ""
	for len(username) == 0 {
		user, err := askInput("Username: ", reader)
		if err != nil {
			continue
		}
		username = user
	}
	for len(password) < 5 {
		pass, err := askPassword("Password: ")
		if err != nil {
			continue
		}
		password = pass
	}
	runCmd(fmt.Sprintf("useradd -m -G sudo,docker -s /usr/bin/zsh %[1]v", username), verbose)
	runCmd(fmt.Sprintf("echo \"%[1]v:%[1]v\" | chpasswd", username), verbose)

	runCmd(fmt.Sprintf("cp -r /root/.zshrc /home/%[1]v/.zshrc", username), verbose)
	runCmd(fmt.Sprintf("chown -R %[1]v:%[1]v /home/%[1]v/.zshrc", username), verbose)
	runCmd(fmt.Sprintf("cp -r /root/.antigen.zsh /home/%[1]v/.antigen.zsh", username), verbose)
	runCmd(fmt.Sprintf("chown -R %[1]v:%[1]v /home/%[1]v/.antigen.zsh", username), verbose)
	runCmd(fmt.Sprintf("su - %[1]v -c \"curl -fsSL https://bun.sh/install | bash\"", username), verbose)

	sshDir := fmt.Sprintf("/home/%[1]v/.ssh", username)
	_, err = os.Stat(sshDir)

	if os.IsNotExist(err) {
		err := os.MkdirAll(sshDir, 0700)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println()
	for {
		sshKey, err := askInput("Enter ssh public key: ", reader)
		if err != nil {
			panic(err)
		}

		if len(sshKey) == 0 {
			break
		}
		file, err := os.OpenFile(fmt.Sprintf("%v/authorized_keys", sshDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0744)
		if err != nil {
			continue
		}
		_, err = file.Write([]byte(sshKey + "\n"))
		if err != nil {
			panic(err)
		}
	}

	fmt.Println()

	runCmd(fmt.Sprintf("chown -R %[1]v:%[1]v %[2]v", username, sshDir), verbose)

	fmt.Println(Green + "All Done!" + Reset)
	fmt.Println()
}

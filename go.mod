module sounding

go 1.16

require (
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-shellwords v1.0.12
	github.com/schollz/progressbar/v3 v3.8.2
	github.com/spf13/pflag v1.0.5
	github.com/valyala/fastjson v1.6.3
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	gopkg.in/validator.v2 v2.0.0-20210331031555-b37d688a7fb0
)

replace (
	github.com/schollz/progressbar/v3 => github.com/oerlikon/progressbar/v3 v3.8.2-patch2
	github.com/valyala/fastjson => github.com/oerlikon/fastjson v1.6.3-patch3
	gopkg.in/yaml.v2 => github.com/oerlikon/yaml/v2 v2.4.0-patch1
)

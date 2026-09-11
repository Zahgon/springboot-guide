// Command bean-validation-demo is the Go port of the Spring Boot application
// com.example.beanvalidationdemo.BeanValidationDemoApplication.
//
// 微信搜 JavaGuide 回复"面试突击"即可免费领取个人原创的 Java 面试手册
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// defaultPort is Spring Boot's default `server.port`. The original's
// application.properties is empty, so nothing overrides it.
const defaultPort = "8080"

func main() {
	port := resolvePort(os.Args[1:], os.Getenv("SERVER_PORT"))
	server := &http.Server{Addr: ":" + port, Handler: newRouter()}
	fmt.Printf("Tomcat started on port(s): %s (http) with context path ''\n", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// resolvePort reads the port from a `--server.port=` argument, then from
// SERVER_PORT, then falls back to Spring Boot's default — the same precedence
// Spring Boot applies to command-line arguments over environment variables.
func resolvePort(args []string, environment string) string {
	const flag = "--server.port="
	for _, argument := range args {
		if value, ok := strings.CutPrefix(argument, flag); ok && value != "" {
			return value
		}
	}
	if environment != "" {
		return environment
	}
	return defaultPort
}

package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env            string
	UserAddr       string
	UserTimeout    time.Duration
	UserRetries    int
	OrderAddr      string
	OrderTimeout   time.Duration
	OrderRetries   int
	ProductAddr    string
	ProductTimeout time.Duration
	ProductRetries int
	HttpAddress    string
	HttpTimeout    time.Duration
	IdleTimeout    time.Duration
}

const (
	defaultTimeout = 2 * time.Second
	defaultRetries = 3
)

func MustLoad() *Config {
	env := os.Getenv("ENV")
	if env == "" {
		log.Fatal("ENV is not set")
	}

	httpAddress := os.Getenv("HTTP_ADDRESS")
	if env == "" {
		log.Fatal("HTTP_ADDRESS is not set")
	}

	httpTimeoutStr := os.Getenv("HTTP_TIMEOUT")
	httpTimeout := setTimeout(httpTimeoutStr)

	idleTimeoutStr := os.Getenv("IDLE_TIMEOUT")
	idleTimeout := setTimeout(idleTimeoutStr)

	userAddr := os.Getenv("USER_ADDR")
	if userAddr == "" {
		log.Fatal("FATAL: USER_ADDR is not set")
	}

	userTimeoutStr := os.Getenv("USER_TIMEOUT")
	userTimeout := setTimeout(userTimeoutStr)

	userRetriesStr := os.Getenv("USER_RETRIES")
	userRetries := setRetries(userRetriesStr)

	orderAddr := os.Getenv("ORDER_ADDR")
	if orderAddr == "" {
		log.Fatal("FATAL: ORDER_ADDR is not set")
	}

	orderTimeoutStr := os.Getenv("ORDER_TIMEOUT")
	orderTimeout := setTimeout(orderTimeoutStr)

	orderRetriesStr := os.Getenv("ORDER_RETRIES")
	orderRetries := setRetries(orderRetriesStr)

	productAddr := os.Getenv("PRODUCT_ADDR")
	if productAddr == "" {
		log.Fatal("FATAL: PRODUCT_ADDR is not set")
	}

	productTimeoutStr := os.Getenv("PRODUCT_TIMEOUT")
	productTimeout := setTimeout(productTimeoutStr)

	productRetriesStr := os.Getenv("PRODUCT_RETRIES")
	productRetries := setRetries(productRetriesStr)

	return &Config{
		Env:            env,
		HttpAddress:    httpAddress,
		HttpTimeout:    httpTimeout,
		IdleTimeout:    idleTimeout,
		UserAddr:       userAddr,
		UserTimeout:    userTimeout,
		UserRetries:    userRetries,
		OrderAddr:      orderAddr,
		OrderTimeout:   orderTimeout,
		OrderRetries:   orderRetries,
		ProductAddr:    productAddr,
		ProductTimeout: productTimeout,
		ProductRetries: productRetries,
	}
}

func setTimeout(strTimeout string) time.Duration {
	var timeout time.Duration
	if strTimeout == "" {
		timeout := defaultTimeout
		log.Printf("INFO: USER_TIMEOUT not set, using default value: %s", timeout.String())
	} else {
		var err error
		timeout, err = time.ParseDuration(strTimeout)
		if err != nil {
			log.Fatalf("FATAL: Invalid format for USER_TIMEOUT ('%s'): %v", strTimeout, err)
		}
	}
	return timeout
}

func setRetries(strRetries string) int {
	var retries int
	if strRetries == "" {
		retries = defaultRetries
		log.Printf("INFO: USER_RETRIES not set, using default value: %d", retries)
	} else {
		var err error
		retries, err = strconv.Atoi(strRetries)
		if err != nil {
			log.Fatalf("FATAL: Invalid format for USER_RETRIES ('%s'): %v", strRetries, err)
		}
		if retries < 0 {
			log.Fatalf("FATAL: USER_RETRIES must be a non-negative integer, got: %d", retries)
		}
	}
	return retries
}

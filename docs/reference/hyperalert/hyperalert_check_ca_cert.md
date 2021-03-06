---
title: Check Ca Cert
menu:
  product_searchlight_{{ .version }}:
    identifier: hyperalert-check-ca-cert
    name: Check Ca Cert
    parent: hyperalert-cli
product_name: searchlight
section_menu_id: reference
menu_name: product_searchlight_{{ .version }}
---
## hyperalert check_ca_cert

Check Certificate expire date

### Synopsis

Check Certificate expire date

```
hyperalert check_ca_cert [flags]
```

### Options

```
  -c, --critical duration   Remaining duration for critical state. [Default: 120h] (default 120h0m0s)
  -h, --help                help for check_ca_cert
  -w, --warning duration    Remaining duration for warning state. [Default: 360h] (default 360h0m0s)
```

### Options inherited from parent commands

```
      --alsologtostderr                  log to standard error as well as files
      --bypass-validating-webhook-xray   if true, bypasses validating webhook xray checks
      --context string                   Use the context in kubeconfig
      --icinga.checkInterval int         Icinga check_interval in second. [Format: 30, 300] (default 30)
      --kubeconfig string                Path to kubeconfig file with authorization information (the master location is set by the master flag).
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files
      --stderrthreshold severity         logs at or above this threshold go to stderr
      --use-kubeapiserver-fqdn-for-aks   if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522 (default true)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```

### SEE ALSO

* [hyperalert](/docs/reference/hyperalert/hyperalert.md)	 - AppsCode Icinga2 plugin



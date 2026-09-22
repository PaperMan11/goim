# =============================================================================
# GoIM Makefile
#
# 服务清单（11 个）：
#   应用层 : api msggateway cron msgtransfer push   （入口 ./im-<name>/cmd）
#   RPC 层 : auth conversation group msg relation user（入口 ./im-rpc/<name>）
#
# 注意：各服务配置中的日志路径为相对路径（如 logs/im-api），因此启动时
#       工作目录必须为服务目录，本文件已通过 Start-Process -WorkingDirectory
#       （Windows）/ cd（类 Unix）处理。
#
# Windows 下需要先安装 make：
#   scoop install make    或    choco install make
# 进程托管通过 PowerShell（Start-Process / Stop-Process）实现，
# 因此无论 make 使用 cmd 还是 sh 作为 recipe shell 均可正常工作。
# =============================================================================

GO        := go
BIN_DIR   := bin
LOG_DIR   := logs
RUN_DIR   := run

# 必须在二进制目标列表之前确定，因为 := 为立即展开
ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

# 应用层服务（二进制名 = 服务名）
APPS := api msggateway cron msgtransfer push
# RPC 服务（二进制名统一加 rpc- 前缀）
RPCS := auth conversation group msg relation user

ALL_SVC := $(APPS) $(RPCS)

# 启动顺序：RPC 优先，消费者次之，API/网关最后；停止顺序相反
START_ORDER := $(RPCS) msgtransfer push cron api msggateway
STOP_ORDER  := msggateway api cron push msgtransfer $(RPCS)

APP_BINS := $(addprefix $(BIN_DIR)/,$(addsuffix $(EXE),$(APPS)))
RPC_BINS := $(addprefix $(BIN_DIR)/rpc-,$(addsuffix $(EXE),$(RPCS)))

SVC_LOG_DIRS := $(addsuffix /logs,$(addprefix im-,$(APPS))) \
                $(addsuffix /logs,$(addprefix im-rpc/,$(RPCS))) \
                $(LOG_DIR)

# -----------------------------------------------------------------------------
# 平台相关定义
# -----------------------------------------------------------------------------
ifeq ($(OS),Windows_NT)

# 后台启动：参数 = 二进制路径, 日志名, 工作目录, 配置文件
define run
	powershell -NoProfile -Command "(Start-Process -FilePath '$(CURDIR)/$(1)' -ArgumentList '-f','$(4)' -WorkingDirectory '$(CURDIR)/$(3)' -RedirectStandardOutput '$(CURDIR)/$(LOG_DIR)/$(2).out.log' -RedirectStandardError '$(CURDIR)/$(LOG_DIR)/$(2).err.log' -WindowStyle Hidden -PassThru).Id | Set-Content -Encoding ascii '$(CURDIR)/$(RUN_DIR)/$(2).pid'"
endef

# 停止：按二进制路径模式匹配，杀掉所有同类进程（含多次启动的残留）
define kill
	-powershell -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { $$_.CommandLine -like '*$(BIN_DIR)/$(1)*' } | ForEach-Object { Stop-Process -Id $$_.ProcessId -Force -ErrorAction SilentlyContinue }; Remove-Item '$(CURDIR)/$(RUN_DIR)/$(1).pid' -Force -ErrorAction SilentlyContinue"
endef

# 查询状态：按二进制路径模式匹配
define ps
	@powershell -NoProfile -Command "$$procs = Get-CimInstance Win32_Process | Where-Object { $$_.CommandLine -like '*$(BIN_DIR)/$(1)*' }; if ($$procs) { $$pids = ($$procs | ForEach-Object { $$_.ProcessId }) -join ' '; '$(1): running (pid ' + $$pids + ')' } else { '$(1): stopped' }"
endef

# 跟踪 stdout 日志：参数 = 日志名
define follow
	powershell -NoProfile -Command "Get-Content -Path '$(CURDIR)/$(LOG_DIR)/$(1).out.log' -Tail 200 -Wait"
endef

define clean_bin
	powershell -NoProfile -Command "Remove-Item -Recurse -Force '$(CURDIR)/$(BIN_DIR)','$(CURDIR)/$(RUN_DIR)' -ErrorAction SilentlyContinue"
endef

define clean_logs
	powershell -NoProfile -Command "$(foreach d,$(SVC_LOG_DIRS),Remove-Item -Recurse -Force '$(CURDIR)/$(d)' -ErrorAction SilentlyContinue;)"
endef

define mkdir
	powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(CURDIR)/$(1)' | Out-Null"
endef
else

define run
	@cd $(3) && $(CURDIR)/$(1) -f $(4) >> $(CURDIR)/$(LOG_DIR)/$(2).out.log 2>&1 & echo $$! > $(CURDIR)/$(RUN_DIR)/$(2).pid
endef

define kill
	-@pkill -f '$(CURDIR)/$(BIN_DIR)/$(1)' 2>/dev/null || true; rm -f $(CURDIR)/$(RUN_DIR)/$(1).pid
endef

define ps
	@pids=`pgrep -f '$(CURDIR)/$(BIN_DIR)/$(1)' 2>/dev/null`; if [ -n "$$pids" ]; then echo "$(1): running (pid $$pids)"; else echo "$(1): stopped"; fi
endef

define follow
	tail -n 200 -F $(CURDIR)/$(LOG_DIR)/$(1).out.log
endef

define clean_bin
	rm -rf $(BIN_DIR) $(RUN_DIR)
endef

define clean_logs
	rm -rf $(SVC_LOG_DIRS)
endef

define mkdir
	mkdir -p $(1)
endef
endif

# -----------------------------------------------------------------------------
# 每个服务的启动元数据（供 start-<svc> 规则使用）
# 参数：SBIN=二进制, SNAME=日志/pid 名, SWORKDIR=工作目录, SCFG=配置文件
# -----------------------------------------------------------------------------
$(foreach s,$(APPS),\
	$(eval start-$(s): SBIN := $(BIN_DIR)/$(s)$(EXE)) \
	$(eval start-$(s): SNAME := $(s)) \
	$(eval start-$(s): SWORKDIR := im-$(s)) \
	$(eval start-$(s): SCFG := etc/$(s).yml) \
	$(eval stop-$(s) restart-$(s) status-$(s) logs-$(s): SNAME := $(s)))

$(foreach s,$(RPCS),\
	$(eval start-$(s): SBIN := $(BIN_DIR)/rpc-$(s)$(EXE)) \
	$(eval start-$(s): SNAME := rpc-$(s)) \
	$(eval start-$(s): SWORKDIR := im-rpc/$(s)) \
	$(eval start-$(s): SCFG := etc/$(s).yaml) \
	$(eval stop-$(s) restart-$(s) status-$(s) logs-$(s): SNAME := rpc-$(s)))

# -----------------------------------------------------------------------------
# 目标声明
# -----------------------------------------------------------------------------
.PHONY: all help build start stop restart status dirs test vet fmt tidy proto clean clean-logs clean-all
.PHONY: $(addprefix build-,$(ALL_SVC)) $(addprefix start-,$(ALL_SVC)) \
        $(addprefix stop-,$(ALL_SVC))  $(addprefix restart-,$(ALL_SVC)) \
        $(addprefix status-,$(ALL_SVC)) $(addprefix logs-,$(ALL_SVC))

.DEFAULT_GOAL := help

all: build

# ----- 构建 ------------------------------------------------------------------

# 应用层：bin/<name>   <- ./im-<name>/cmd  ($* = stem，即服务名)
$(APP_BINS): $(BIN_DIR)/%$(EXE): | $(BIN_DIR)
	$(GO) build -o $@ ./im-$*/cmd
	@chmod +x $@

# RPC 层：bin/rpc-<name> <- ./im-rpc/<name>
$(RPC_BINS): $(BIN_DIR)/rpc-%$(EXE): | $(BIN_DIR)
	$(GO) build -o $@ ./im-rpc/$*
	@chmod +x $@

build: $(APP_BINS) $(RPC_BINS)
	@echo build complete: $(BIN_DIR)/

$(addprefix build-,$(APPS)): build-%: $(BIN_DIR)/%$(EXE)
$(addprefix build-,$(RPCS)): build-%: $(BIN_DIR)/rpc-%$(EXE)

# ----- 启动 / 停止 -----------------------------------------------------------

$(addprefix start-,$(ALL_SVC)): start-%: | dirs
	@echo [+] starting $(SNAME) ...
	$(call run,$(SBIN),$(SNAME),$(SWORKDIR),$(SCFG))

$(addprefix stop-,$(ALL_SVC)): stop-%:
	@echo [-] stopping $(SNAME) ...
	$(call kill,$(SNAME))

$(addprefix status-,$(ALL_SVC)): status-%:
	$(call ps,$(SNAME))

$(addprefix logs-,$(ALL_SVC)): logs-%:
	$(call follow,$(SNAME))

$(addprefix restart-,$(ALL_SVC)): restart-%: stop-% start-%

# 先构建再按顺序启动全部服务
start: build $(addprefix start-,$(START_ORDER))
	@echo all services started. pid files in $(RUN_DIR)/, stdout logs in $(LOG_DIR)/

# 按相反顺序停止全部服务
stop: $(addprefix stop-,$(STOP_ORDER))
	@echo all services stopped.

restart: stop start

status: $(addprefix status-,$(ALL_SVC))

# ----- Go 工具链 -------------------------------------------------------------

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -s -w .

tidy:
	$(GO) mod tidy

# 由 proto 重新生成 gRPC 代码（需要 protoc / protoc-gen-go / protoc-gen-go-grpc）
proto:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File '$(CURDIR)/pkg/protocol/gen_grpc.ps1'
else
	bash pkg/protocol/gen_grpc.sh
endif

# ----- 清理 ------------------------------------------------------------------

clean:
	$(call clean_bin)
	@echo cleaned $(BIN_DIR)/ and $(RUN_DIR)/

clean-logs:
	$(call clean_logs)
	@echo cleaned log directories

clean-all: clean clean-logs

# ----- 目录 ------------------------------------------------------------------

dirs: $(BIN_DIR) $(LOG_DIR) $(RUN_DIR)

$(BIN_DIR) $(LOG_DIR) $(RUN_DIR):
	$(call mkdir,$@)

# ----- 帮助 ------------------------------------------------------------------

help:
	@echo GoIM Makefile - available targets:
	@echo   build            Build all 11 binaries into $(BIN_DIR)/
	@echo   build-SVC        Build a single service, e.g. build-api / build-msg
	@echo   start            Build and start all services [background, pid in $(RUN_DIR)/]
	@echo   stop             Stop all services
	@echo   restart          Stop then start all services
	@echo   status           Show running status of all services
	@echo   start-SVC        Start one service: api msggateway cron msgtransfer push
	@echo                    auth conversation group msg relation user
	@echo   stop-SVC         Stop one service
	@echo   restart-SVC      Restart one service
	@echo   logs-SVC         Follow one service stdout log [$(LOG_DIR)/SVC.out.log]
	@echo   test             Run go test ./...
	@echo   vet              Run go vet ./...
	@echo   fmt              Run gofmt -s -w .
	@echo   tidy             Run go mod tidy
	@echo   proto            Regenerate gRPC code from proto files
	@echo   clean            Remove $(BIN_DIR)/ and $(RUN_DIR)/
	@echo   clean-logs       Remove all service log directories
	@echo   clean-all        clean + clean-logs

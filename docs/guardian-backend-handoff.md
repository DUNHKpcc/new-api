# Guardian 服务订单：后端交接文档

> 本地二开功能，只在 dev 分支。前端已完成（`web/src/features/guardian/`），本文是前端已经在用的接口约定。后端按本文实现后，前端不用再改。
>
> dev 分支的 AGENTS.md 默认不允许改后端业务、支付和数据库结构。这次的新表和新接口由项目所有者本人批准并实现；提交说明里请注明这一点。

## 1. 范围

- **前台** `/guardian`：宣传 Claude-Guardian Skill。首屏显示的 GitHub star 数由浏览器直接请求 GitHub 公开接口获取，后端不用管。
- **服务页** `/guardian/services`：环境配置、部署类人工服务的目录、下单和服务流程。
- **我的订单** `/guardian/orders`：需要登录，用户在这里看自己的订单和交付进度。
- **后台** `/guardian-admin`：管理员编辑服务目录（名称、规格、文案、价格、上下架），推进订单状态，写交付备注。
- **付款**：只走现有的**易支付**，下单后跳转到易支付收银台。不走钱包余额，不接 Stripe、Creem、Waffo。
- **退款**：在易支付商户后台手动退款，系统里只把订单标成"已退款"。不调用退款接口，也不碰额度计费代码（`.agents/rules/billing.md` 管不到这里）。
- **不发通知**：不发邮件，也不推送。

## 2. 数据表

两张表都要通过 `AutoMigrate` 加入主库迁移。按 AGENTS.md 的要求，要在 SQLite、MySQL、PostgreSQL 上各自验证：新库能建表、旧库能升级、重复启动不报错。

### 2.1 `guardian_services`：服务目录

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int 主键 | 由 GORM 生成 |
| `title` | varchar(128) not null | 服务名称 |
| `subtitle` | varchar(255) | 一句话简介 |
| `description` | text | 详细文案，纯文本，段落之间空一行 |
| `features` | text | 规格清单。数据库里存 JSON 字符串数组，**接口里返回数组**。最多 12 条，每条不超过 120 字 |
| `price_cents` | bigint not null | 人民币价格，单位是**分**，取值 1 到 10,000,000 |
| `delivery_note` | varchar(128) | 交付周期，例如"1–3 个工作日" |
| `badge` | varchar(32) | 可选角标，例如"推荐"，空字符串表示不显示 |
| `route` | varchar(16) | 适用路线，只能是 `any`、`local`、`vm`、`vps` 之一 |
| `enabled` | bool | 是否上架。**不要写 `gorm:"default:true"`**，默认值在代码里设置（见 AGENTS.md 的布尔默认值规则） |
| `sort_order` | int | 排序，数字越小越靠前 |
| `created_at` / `updated_at` | bigint | Unix 秒 |

### 2.2 `guardian_service_orders`：服务订单

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | int 主键 | |
| `trade_no` | varchar(64) unique index | 格式 `SVCUSR{userId}NO{6位随机}{unix秒}`，和现有的 `SUBUSR...` 命名一致 |
| `user_id` | int index | |
| `service_id` | int index | |
| `service_title` | varchar(128) | 下单时的服务名**快照** |
| `amount_cents` | bigint | 下单时的价格**快照**，单位分。以后改价不影响已有订单 |
| `payment_method` | varchar(32) | 易支付的支付类型，例如 `alipay`、`wxpay` |
| `status` | varchar(24) index | 见第 3 节 |
| `route` | varchar(16) | `local`、`vm`、`vps`、`unsure` 之一 |
| `os` | varchar(32) | `macos`、`windows`、`linux`、`other` 之一 |
| `contact_type` | varchar(16) | `telegram`、`qq`、`wechat`、`email` 之一 |
| `contact` | varchar(128) | 联系方式，只有本人和管理员能看到 |
| `note` | text | 用户备注，最多 1000 字 |
| `admin_note` | text | 管理员给用户的留言，**用户能看到**，最多 2000 字 |
| `provider_payload` | text | 易支付回调的原始校验结果，只给管理员看，前端没有用到 |
| `created_at` / `paid_at` / `delivered_at` / `updated_at` | bigint | Unix 秒，没有发生过的时间填 0 |

## 3. 订单状态

| 状态 | 怎么进入 |
| --- | --- |
| `pending_payment` | 下单成功 |
| `paid` | 易支付回调或跳回校验成功，**而且实付金额和订单一致** |
| `scheduled` | 管理员排期 |
| `in_progress` | 管理员开始交付 |
| `delivered` | 管理员确认交付完成，同时写入 `delivered_at` |
| `expired` | 待付款超过 2 小时。可以在用户和管理员查询列表时顺手把过期订单标掉，也可以用定时任务，二选一 |
| `cancelled` | 管理员取消一笔还没付款的订单 |
| `refunded` | 管理员在易支付后台退款之后手动标记 |

管理员能做的状态变更**只有下面这些**，其他变更一律返回 `success:false`：

- `pending_payment` 可以改成 `cancelled`
- `paid` 可以改成 `scheduled`、`in_progress`、`refunded`
- `scheduled` 可以改成 `in_progress`、`refunded`
- `in_progress` 可以改成 `delivered`、`refunded`
- `delivered` 可以改成 `refunded`

只改 `admin_note`、不改状态（提交原来的状态）始终允许。

## 4. 接口

所有接口都使用项目统一的响应格式 `{ "success": bool, "message": string, "data": ... }`，前端通过 `requireServerSuccess` 读取。

### 4.1 公开接口

`GET /api/guardian/services`，不需要登录。

```json
{
  "success": true,
  "data": {
    "services": [GuardianService],
    "pay_methods": [{ "type": "alipay", "name": "支付宝" }],
    "purchase_enabled": true
  }
}
```

- `services`：只返回已上架的服务，按 `sort_order`、`id` 升序。
- `pay_methods`：取 `operation_setting.PayMethods` 里属于易支付的项，排除 `stripe`、`creem`、`waffo` 等类型。
- `purchase_enabled`：易支付客户端已配置（`GetEpayClient() != nil`）、支付合规已确认（`IsPaymentComplianceConfirmed()`），并且 `pay_methods` 不为空，三者同时满足才是 `true`。为 `false` 时，前端照常展示服务，但下单按钮会变成"暂不可下单"。

`GuardianService` 的结构：

```json
{
  "id": 1,
  "title": "VPS 开发机部署",
  "subtitle": "把远程服务器配成稳定的 Claude Code 开发机",
  "description": "第一段……\n\n第二段……",
  "features": ["Ubuntu 24.04 环境一致", "Xvfb + noVNC 远程桌面"],
  "price_cents": 59900,
  "delivery_note": "1–3 个工作日",
  "badge": "推荐",
  "route": "vps",
  "enabled": true,
  "sort_order": 10,
  "created_at": 1761500000,
  "updated_at": 1761500000
}
```

### 4.2 用户接口（`middleware.UserAuth()`）

`POST /api/guardian/orders`，挂 `middleware.CriticalRateLimit()`。

请求：

```json
{
  "service_id": 1,
  "payment_method": "alipay",
  "route": "vps",
  "os": "macos",
  "contact_type": "telegram",
  "contact": "@someone",
  "note": "可选，最多 1000 字"
}
```

服务端校验，前端也做了同样的校验，但不能依赖前端：

1. 先调用 `requirePaymentCompliance(c)`。
2. 服务存在并且已上架，`price_cents` 至少为 1。
3. `operation_setting.ContainsPayMethod(payment_method)` 为真。
4. `route`、`os`、`contact_type` 都在第 2.2 节列出的取值里。`contact` 去掉首尾空格后长度为 1 到 128。`note` 不超过 1000 字。
5. 同一用户同时最多挂 3 笔 `pending_payment` 订单，超过就返回 `success:false`，message 写"待付款订单过多，请先完成或等待过期"。
6. 价格换算：`Money = 分 / 100`，保留两位小数，不经过 float 乘法。例如 `fmt.Sprintf("%d.%02d", cents/100, cents%100)`。
7. 照抄 `SubscriptionRequestEpay` 的写法调用 `client.Purchase`。`Name` 写 `SVC:{title}`；`NotifyUrl` 是 `{callback}/api/guardian/epay/notify`；`ReturnUrl` 是 `{callback}/api/guardian/epay/return`。如果拉起支付失败，把订单标成 `expired`。

响应：

```json
{
  "success": true,
  "data": {
    "trade_no": "SVCUSR12NOab12cd1761500000",
    "pay_url": "https://pay.example.com/submit.php",
    "pay_params": { "pid": "...", "sign": "...", "...": "..." }
  }
}
```

前端会把 `pay_params` 当作隐藏字段，用 POST 表单提交到 `pay_url`。

`GET /api/guardian/orders/self?p=1&page_size=10` 返回当前用户的订单，按 `id` 倒序，`page_size` 最大 50。

```json
{ "success": true, "data": { "items": [GuardianOrder], "total": 3 } }
```

`GuardianOrder`（用户视角）的字段：`id`、`trade_no`、`service_id`、`service_title`、`amount_cents`、`payment_method`、`status`、`route`、`os`、`contact_type`、`contact`、`note`、`admin_note`、`created_at`、`paid_at`、`delivered_at`、`updated_at`。**不包含** `user_id` 和 `provider_payload`。

### 4.3 管理员接口（`middleware.AdminAuth()`）

| 方法和路径 | 请求 | 返回的 `data` |
| --- | --- | --- |
| `GET /api/guardian/admin/services` | 无 | `GuardianService[]`，包括已下架的 |
| `POST /api/guardian/admin/services` | `ServicePayload` | `GuardianService` |
| `PUT /api/guardian/admin/services/:id` | `ServicePayload`，整体替换 | `GuardianService` |
| `GET /api/guardian/admin/orders?p=1&page_size=20&status=&keyword=` | 无 | `{ items: AdminGuardianOrder[], total }` |
| `PUT /api/guardian/admin/orders/:id` | `{ "status": "...", "admin_note": "..." }` | `AdminGuardianOrder` |

- `ServicePayload` 包含 `title`、`subtitle`、`description`、`features`（字符串数组）、`price_cents`、`delivery_note`、`badge`、`route`、`enabled`、`sort_order`。校验规则同第 2.1 节；所有字符串去掉首尾空格；`features` 里的空行要过滤掉。
- 列表筛选：`status` 为空表示全部状态。`keyword` 按 `trade_no`、用户名、`contact` 做模糊匹配，用 GORM 的 `LIKE`，不要用任何数据库特有的写法。
- `AdminGuardianOrder` 在 `GuardianOrder` 的基础上多两个字段：`user_id` 和 `username`。
- 服务**不提供删除接口**，只能下架，保证历史订单能对上原来的服务。

### 4.4 易支付回调

照着 `SubscriptionEpayNotify` 和 `SubscriptionEpayReturn` 写，换成新的路由：

```go
apiRouter.POST("/guardian/epay/notify", anonymousRequestBodyLimit, controller.GuardianEpayNotify)
apiRouter.GET("/guardian/epay/notify", controller.GuardianEpayNotify)
apiRouter.GET("/guardian/epay/return", controller.GuardianEpayReturn)
apiRouter.POST("/guardian/epay/return", anonymousRequestBodyLimit, controller.GuardianEpayReturn)
```

完成订单的函数（例如 `model.CompleteGuardianOrder(tradeNo, payload, paidMoney)`）必须做到这几点：

1. 在 `LockOrder(tradeNo)` 之内运行，并开数据库事务，用 `lockForUpdate(tx)` 锁住订单行。
2. 找不到订单，返回 not found。
3. 订单已经不是 `pending_payment`，**直接返回成功**，保证幂等：易支付会重复回调，跳回和回调也可能同时到达。
4. 把 `verifyInfo.Money` 换算成分，和 `amount_cents` 比较。不一致就拒绝，订单保持不变，并记一条 `logger.LogError`。
5. 把订单更新为 `status=paid`，写入 `paid_at` 和 `provider_payload`，并在 `payment_method` 写入实际支付类型 `verifyInfo.Type`。

浏览器跳回的地址固定如下，前端订单页靠这个 `pay` 参数显示提示条：

| 情况 | 跳转到 |
| --- | --- |
| 付款成功并完成订单 | `paymentReturnPath("/guardian/orders?pay=success")` |
| 交易还没成功 | `paymentReturnPath("/guardian/orders?pay=pending")` |
| 校验失败或出错 | `paymentReturnPath("/guardian/orders?pay=fail")` |

**完全不需要改动**现有的 `topup_*.go`、`subscription_payment_*.go` 和 Stripe、Creem、Waffo 的回调。

## 5. 前端已经替你做好的部分

- 顶部导航默认显示 "Guardian" 入口。想隐藏的话，在 `HeaderNavModules` 里加 `"guardian": false`，前端会读取这个值，后端不用改。
- 后台侧边栏的 Admin 分组下新增了 "Guardian 服务"（`/guardian-admin`），只有管理员能看到。
- 下单表单上写明了"不要填写密码、密钥或订阅链接"，远程访问方式在线下交换。后端只需按长度校验 `contact` 和 `note`。
- 未登录的访客点"预约"会先去登录，登录后回到 `/guardian/services?service={id}` 并自动打开下单弹窗。这个流程纯前端，后端不用管。
- 接口失败时，前端会把响应里的 `message` 直接以提示框显示给用户，所以 `message` 要写成用户能看懂的话。

**前端以哪些文件为准**：接口路径和参数在 `web/src/features/guardian/api.ts`，字段类型在 `web/src/features/guardian/types.ts`，后台允许的状态变更在 `web/src/features/guardian/constants.ts` 的 `GUARDIAN_ADMIN_TRANSITIONS`。如果后端实现时想改约定，请同时改这三个文件和本文。

**页面文案里已经写下的承诺**，后端实现要和它们一致；要改的话，前端文案也得一起改：

- 订单页："未付款的订单两小时后过期"，对应第 3 节的 `expired`。
- 常见问题："退款由客服处理，处理完成后订单会标记为已退款"，对应第 3 节的 `refunded`。
- 常见问题："订单会在我的订单里显示为已付款，之后依次进入已排期、交付中、已交付"，对应第 3 节的状态流转。

## 6. 本地联调

后端和前端开发服务器默认都用 3000 端口，所以后端要换一个端口：

```bash
# 后端（仓库根目录）
go run . --port 3001

# 前端（web/），把开发服务器的接口请求代理到 3001
cd web
VITE_REACT_APP_SERVER_URL=http://localhost:3001 bun run dev
```

然后打开 `http://localhost:3000/guardian`。本机没有装 bun 的话，可以用 `mise exec bun@latest -- bun run dev` 临时运行。

联调时建议按这个顺序走一遍：

1. 在后台 `/guardian-admin` 的"服务目录"里新建一项服务，回到 `/guardian/services` 确认能看到。
2. 下单，确认浏览器跳到易支付收银台，订单页出现一笔"待付款"。
3. 付款后，确认易支付回调把订单改成"已付款"，并跳回 `/guardian/orders?pay=success`。
4. 在后台把订单推进到"已排期 → 交付中 → 已交付"，并写一条留言，确认用户的订单页能看到进度和留言。

## 7. 建议的测试

按 AGENTS.md 的要求，测试集中写在**一个**测试文件里（例如 `model/guardian_service_test.go`），用 testify：

- 下单：服务已下架、价格为 0、支付方式不存在、待付款超过 3 笔，都应被拒绝。
- 回调：金额不一致时拒绝；同一笔订单回调两次，只完成一次，第二次直接返回成功；订单号不存在时返回 not found。
- 状态机：第 3 节以外的状态变更全部被拒绝；只改 `admin_note` 允许。
- 用户订单列表不返回别人的订单，也不包含 `provider_payload`。
- 迁移：分别在 SQLite、MySQL、PostgreSQL 上验证新库建表、旧库升级、重复启动两次，并记录数据库版本和执行的命令。

## 8. 后端实现与验证（2026-10-28）

用户已明确授权按本交接文档实现后端、支付接入和数据库改动。本次新增 `model/guardian.go`、`controller/guardian.go`，接入普通及快速启动迁移与 API 路由；前端接口约定保持不变。Guardian 用户及管理接口使用现有登录会话鉴权，个人访问令牌不开放这些接口。易支付回调使用现有验签与订单锁，金额精确比较，支付签名不进入访问日志。

验证环境为独立本地数据库：SQLite **3.50.4**、MySQL **8.4.11**、PostgreSQL **16.15**。`router/guardian_test.go` 集中覆盖目录编辑、上下架、参数校验、支付合规、支付方式过滤、三笔待付款上限（含并发）、签名与金额校验、重复回调、状态机、两小时过期、用户订单隔离、管理员权限、PAT 拒绝、过期与撤销会话、日志签名脱敏、重复迁移及订单号唯一约束。

已执行并通过：

```bash
go build ./...
go test ./model ./controller ./router
TEST_MYSQL_DSN='root@tcp(127.0.0.1:13316)/guardian_test?parseTime=true&charset=utf8mb4' \
TEST_POSTGRES_DSN='host=127.0.0.1 port=15436 user=postgres dbname=guardian_test sslmode=disable' \
go test ./router -run 'TestGuardianLifecycle|TestAccessTokenRouteRulesCoverEveryDashboardRoute' -count=1 -v
```

三种数据库均验证了新建 Guardian 表与重复迁移。旧库升级额外使用 GitHub 最新发布版 **v1.0.0-rc.46** 的归档源码创建完整数据库并写入代表性用户（余额 123456），随后运行当前版本 `model.InitDB()` 三次，检查用户数据、用户名唯一约束及 Guardian 新表；再在升级后的数据库上运行同一套接口测试，全部通过。迁移验证使用临时程序 `/tmp/guardian-migrate.go`（调用 `model.InitDB()`，关闭连接后再次初始化，最后检查数据与唯一约束）：

```bash
# 在 v1.0.0-rc.46 归档目录运行以下三条，各自创建旧库。
GUARDIAN_SEED=1 TEST_GUARDIAN_SQLITE=/tmp/guardian-upgrade.db go run /tmp/guardian-migrate.go
GUARDIAN_SEED=1 SQL_DSN='root@tcp(127.0.0.1:13316)/guardian_upgrade?parseTime=true&charset=utf8mb4' go run /tmp/guardian-migrate.go
GUARDIAN_SEED=1 SQL_DSN='postgres://postgres@127.0.0.1:15436/guardian_upgrade?sslmode=disable' go run /tmp/guardian-migrate.go
# 在当前源码目录移除 GUARDIAN_SEED=1，分别重跑上述三条验证升级及重复启动。
TEST_GUARDIAN_SQLITE=/tmp/guardian-upgrade.db \
TEST_MYSQL_DSN='root@tcp(127.0.0.1:13316)/guardian_upgrade?parseTime=true&charset=utf8mb4' \
TEST_POSTGRES_DSN='host=127.0.0.1 port=15436 user=postgres dbname=guardian_upgrade sslmode=disable' \
go test ./router -run TestGuardianLifecycle -count=1 -v
```

访问控制依据 [OWASP ASVS 5.0.0 V8](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x17-V8-Authorization.md) 的 8.2.1、8.2.2、8.2.3、8.3.1，以及 [Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)、[Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)：在服务端执行角色、对象和字段访问限制，并复用已有会话有效性校验；相关拒绝与隔离用例已覆盖。这不代表对整个项目的 ASVS 合规认证。

验证使用本地支付签名夹具，未向真实商户发起交易，未验证公网回调可达性，未部署生产，也未执行浏览器视觉测试。独立日志数据库没有结构变化。

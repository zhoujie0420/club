<template>
  <view class="console" :style="safePad">
    <view v-if="!authenticated" class="login-panel">
      <text class="eyebrow">HOLE CLUB / STAFF CONSOLE</text
      ><text class="login-title">今夜，从这里开始。</text
      ><text class="muted">员工运营测试台 · 仅使用虚构客户数据</text>
      <view class="panel login-card"
        ><text class="section-title">进入测试工作台</text
        ><text class="field-label">选择员工身份</text
        ><view class="role-grid"
          ><button
            v-for="person in staffChoices"
            :key="person.id"
            :class="['role-card', { picked: selectedStaff === person.id }]"
            @click="selectedStaff = person.id"
          >
            <b>{{ person.name }}</b
            ><text>{{ person.roleName }}</text>
          </button></view
        ><input
          v-model="code"
          password
          placeholder="输入员工登录口令"
          @confirm="signIn"
        /><button class="primary" :disabled="busy" @click="signIn">
          {{ busy ? "正在验证…" : "进入工作台 →" }}</button
        ><text v-if="error" class="error">{{ error }}</text></view
      >
      <text class="muted"
        >数据保存在服务器，多设备共享。收款仅为线下记录。</text
      ><button class="text-button" @click="openPrivacy">隐私说明</button>
    </view>
    <template v-else>
      <view class="header"
        ><view
          ><text class="eyebrow">HOLE CLUB</text
          ><text class="page-title">{{ titles[tab] }}</text></view
        ><button class="avatar" :style="avatarPad" @click="tab = 'mine'">
          HC
        </button></view
      >
      <view class="connection"
        ><text>{{ error ? "连接异常" : "● 共享测试环境" }}</text
        ><text>{{ state?.businessDate || "—" }} 营业日</text></view
      >
      <view v-if="error" class="error banner"
        >{{ error }}<text @click="refresh">重新加载 ↻</text></view
      >
      <view v-if="!state" class="panel placeholder">{{
        loading ? "正在加载营业数据…" : "暂无数据，请点击重新加载"
      }}</view>
      <template v-if="state">
        <template v-if="tab === 'home'">
          <view class="welcome"
            ><text class="muted">运营概览 / OVERVIEW</text
            ><text>让每一桌，都井然有序。</text></view
          >
          <view class="metrics"
            ><view class="metric featured" @click="showOrders('all')"
              ><text>今日预订</text
              ><b>{{
                todayOrders.filter((o) => o.status !== "cancelled").length
              }}</b
              ><text>本营业日有效订单 ↗</text></view
            ><view class="metric" @click="showOrders('reserved')"
              ><text>待到店</text
              ><b>{{
                todayOrders.filter((o) => o.status === "reserved").length
              }}</b
              ><text>关注客户到店时间</text></view
            ><view class="metric" @click="showOrders('active')"
              ><text>使用中</text
              ><b>{{
                todayOrders.filter((o) =>
                  ["arrived", "serving"].includes(o.status),
                ).length
              }}</b
              ><text>现场服务进行中</text></view
            ><view
              v-if="can('report:revenue') || can('report:own')"
              class="metric"
              @click="showOrders('all')"
              ><text>今日已收</text><b class="money">¥{{ money(revenue) }}</b
              ><text>仅统计已结账订单</text></view
            ></view
          >
          <view class="section-heading"
            ><text>快捷操作</text><text class="muted">QUICK ACTIONS</text></view
          ><view class="quick"
            ><button
              v-if="can('order:create')"
              class="primary"
              @click="startBooking()"
            >
              ＋ 创建预订</button
            ><button @click="tab = 'tables'">查看全场台位 ↗</button></view
          >
          <view class="section-heading"
            ><text>现场待办</text
            ><text class="muted">{{ pending.length }} 项</text></view
          ><view v-if="!pending.length" class="panel empty"
            >当前没有紧急待办，准备迎接下一位客人。</view
          ><view
            v-for="o in pending.slice(0, 5)"
            :key="o.id"
            class="order-card"
            @click="selected = o"
            ><view class="table-icon">{{ o.tableId }}</view
            ><view class="grow"
              ><b>{{ o.customerName }}</b
              ><text class="muted"
                >{{ o.arrivalTime }} · {{ o.guestCount }} 人</text
              ></view
            ><text :class="['badge', o.status]">{{
              labels[o.status]
            }}</text></view
          >
        </template>
        <template v-if="tab === 'tables'">
          <view class="toolbar"
            ><picker
              mode="date"
              :value="tableDate"
              @change="tableDate = $event.detail.value"
              ><button>{{ tableDate }} ▾</button></picker
            ><button @click="refresh">
              {{ loading ? "刷新中…" : "刷新 ↻" }}
            </button></view
          >
          <view class="chips"
            ><button
              v-for="f in tableFilters"
              :key="f.key"
              :class="{ chosen: tableFilter === f.key }"
              @click="tableFilter = f.key"
            >
              {{ f.label }} {{ tableCount(f.key) }}
            </button></view
          >
          <view class="floor panel"
            ><view class="stage">DJ STAGE / 舞台</view
            ><template v-for="zone in ['VIP', '卡座']" :key="zone"
              ><view class="zone-heading"
                >{{ zone === "VIP" ? "VIP LOUNGE" : "MAIN FLOOR" }}
                <text>{{ zone }}区</text></view
              ><view class="seats"
                ><button
                  v-for="t in visibleTables.filter((t) => t.zone === zone)"
                  :key="t.id"
                  :class="['seat', tableStatus(t.id)]"
                  @click="selectedTable = t"
                >
                  <b>{{ t.id }}</b
                  ><text>{{ labels[tableStatus(t.id)] }}</text
                  ><text class="capacity">{{ t.capacity }} 人</text>
                </button></view
              ><view v-if="zone === 'VIP'" class="dance"
                >DANCE FLOOR / 舞池</view
              ></template
            ><view class="bar">BAR / 吧台</view></view
          >
          <text class="footnote"
            >{{ state.tables.length }} 个台位 · 有变更时同步 · 更新于
            {{ updatedTime }}</text>
          >
        </template>
        <template v-if="tab === 'orders'">
          <view class="toolbar"
            ><input
              v-model="query"
              class="search"
              placeholder="搜索姓名、手机号、台号或订单号"
            /><button
              v-if="can('order:create')"
              class="small-primary"
              @click="startBooking()"
            >
              ＋
            </button></view
          >
          <view class="chips"
            ><button
              v-for="f in orderFilters"
              :key="f.key"
              :class="{ chosen: orderFilter === f.key }"
              @click="orderFilter = f.key"
            >
              {{ f.label }}
            </button></view
          ><view class="toolbar"
            ><text class="muted"
              >已加载 {{ filteredOrders.length }} 笔 · 最新创建优先</text
            ><button :disabled="orderLoading" @click="loadOrders(true)">刷新 ↻</button></view
          >
          <view v-if="!filteredOrders.length" class="panel empty"
            >没有符合条件的订单<button
              @click="
                query = '';
                orderFilter = 'all';
              "
            >
              清空筛选
            </button></view
          >
          <view
            v-for="o in filteredOrders"
            :key="o.id"
            class="order-card full"
            @click="selected = o"
            ><view class="row"
              ><view class="table-icon">{{ o.tableId }}</view
              ><view class="grow"
                ><b>{{ o.customerName }} · {{ o.guestCount }} 人</b
                ><text class="muted"
                  >{{ o.date }} {{ o.arrivalTime }}</text
                ></view
              ><text :class="['badge', o.status]">{{
                labels[o.status]
              }}</text></view
            ><view class="card-bottom"
              ><text>{{
                o.items.map((p) => p.name).join("、") || "到店后选购"
              }}</text
              ><b>¥{{ money(total(o)) }}</b></view
            ></view
          >
          <button v-if="orderCursor" class="load-more" :disabled="orderLoading" @click="loadOrders(false)">{{ orderLoading ? "加载中…" : "加载更多历史订单" }}</button><text v-if="orderListError" class="error">{{ orderListError }}</text>
        </template>
        <template v-if="tab === 'mine'"
          ><view class="panel profile"
            ><view class="avatar">HC</view
            ><text class="section-title">{{ state.currentUser.name }}</text
            ><text class="muted"
              >{{ state.currentUser.roleName }} · HOLE CLUB</text
            ></view
          ><view class="panel"
            ><view class="info-row"
              ><text>数据存储</text><b>服务器 SQLite</b></view
            ><view class="info-row"
              ><text>数据同步</text><b>5 秒增量轮询</b></view>
            ><view class="info-row"
              ><text>收款模式</text><b>线下收款记录</b></view
            ><view class="info-row"
              ><text>版本</text><b>0.10 · 财务导出测试版</b></view
            ><view class="info-row"
              ><text>当前权限</text><b>{{ permissionSummary }}</b></view
            ></view
          ><view v-if="canReport" class="panel report-panel"
            ><view class="section-row"><view><text class="section-title">营业报表</text><text class="muted">{{ report?.scope === "own" ? "仅本人业绩" : "全店数据" }}</text></view><button class="text-button" :disabled="reportLoading" @click="loadReport">{{ reportLoading ? "计算中…" : "刷新" }}</button></view
            ><view class="date-range"><picker mode="date" :value="reportFrom" :end="reportTo" @change="reportFrom = $event.detail.value; loadReport()"><view class="field">{{ reportFrom }} ▾</view></picker><text>至</text><picker mode="date" :value="reportTo" :start="reportFrom" :end="state.businessDate" @change="reportTo = $event.detail.value; loadReport()"><view class="field">{{ reportTo }} ▾</view></picker></view
            ><view v-if="report" class="report-grid"><view><text>净营业收入</text><b>¥{{ money(report.totals.revenue) }}</b></view><view><text>已结账订单</text><b>{{ report.totals.completed }}</b></view><view><text>取消订单</text><b>{{ report.totals.cancelled }}</b></view><view><text>平均客单</text><b>¥{{ money(report.totals.completed ? Math.round(report.totals.revenue / report.totals.completed) : 0) }}</b></view><view><text>退款笔数</text><b>{{ report.totals.refunded }}</b></view><view><text>退款金额</text><b>¥{{ money(report.totals.refundAmount) }}</b></view></view
            ><view v-if="report" class="report-days"><view v-for="day in [...report.days].reverse()" :key="day.date"><text>{{ day.date.slice(5) }}</text><text>{{ day.completed }} 单 · {{ day.guests }} 人</text><b>¥{{ money(day.revenue) }}</b></view></view><button v-if="report" class="export-button" @click="exportReport">导出报表 CSV</button
            ><text v-if="reportError" class="error">{{ reportError }}</text></view
          ><view v-if="can('audit:view')" class="panel"
            ><view class="section-row"
              ><text class="section-title">最近操作审计</text
              ><button class="text-button" :disabled="auditLoading" @click="loadAudit">
                {{ auditLoading ? "读取中…" : "刷新" }}
              </button></view
            ><view class="audit-filters"><view class="date-range"><picker mode="date" :value="auditFrom" :end="auditTo" @change="auditFrom = $event.detail.value; loadAudit()"><view class="field">{{ auditFrom }} ▾</view></picker><text>至</text><picker mode="date" :value="auditTo" :start="auditFrom" @change="auditTo = $event.detail.value; loadAudit()"><view class="field">{{ auditTo }} ▾</view></picker></view><view class="filter-pickers"><picker :range="['全部员工', ...staffChoices.map((person) => person.name)]" @change="auditUser = Number($event.detail.value) ? staffChoices[Number($event.detail.value) - 1].id : ''; loadAudit()"><view class="field">{{ staffChoices.find((person) => person.id === auditUser)?.name || '全部员工' }} ▾</view></picker><picker :range="auditCategories.map((item) => item.name)" @change="auditCategory = auditCategories[Number($event.detail.value)].id; loadAudit()"><view class="field">{{ auditCategories.find((item) => item.id === auditCategory)?.name }} ▾</view></picker></view></view
            ><text v-if="auditError" class="error">{{ auditError }}</text
            ><text v-else-if="!auditLogs.length" class="muted">暂无操作记录</text
            ><view v-for="log in auditLogs" :key="log.id" class="event audit-event"
              ><view><text>{{ log.action }}</text><small> · {{ log.userName }}（{{ log.roleName }}）</small></view
              ><text class="muted">{{ log.orderId }} · {{ new Date(log.createdAt).toLocaleString('zh-CN', { hour12: false }) }}</text></view
            ><button v-if="auditLogs.length" class="export-button" @click="exportAudit">导出审计 CSV</button
            ></view
          ><view v-if="can('staff:manage')" class="panel"
            ><view class="section-row"
              ><text class="section-title">员工账号</text
              ><button class="text-button" :disabled="staffLoading" @click="loadStaff">{{ staffLoading ? "读取中…" : "刷新" }}</button></view
            ><view v-for="person in staffAccounts" :key="person.id" class="staff-row"
              ><view><b>{{ person.name }}</b><text>{{ person.roleName }} · {{ person.active ? "已启用" : "已停用" }}</text></view
              ><view class="staff-actions"><button :disabled="busy || !person.active" @click="resetStaffPin(person)">改口令</button><button v-if="person.id !== state.currentUser.id" :class="{ danger: person.active }" :disabled="busy" @click="toggleStaff(person)">{{ person.active ? "停用" : "启用" }}</button></view></view
            ><text class="field-label">新增员工</text
            ><label>员工姓名<input v-model="staffForm.name" maxlength="30" placeholder="例如：销售小陈" /></label
            ><label>员工角色<picker :range="roleChoices.map((r) => r.name)" @change="staffForm.role = roleChoices[Number($event.detail.value)].id"><view class="field">{{ roleChoices.find((r) => r.id === staffForm.role)?.name }} ▾</view></picker></label
            ><label>初始口令<input v-model="staffForm.pin" password maxlength="32" placeholder="至少 6 位，仅当面告知员工" /></label
            ><button class="primary full-button" :disabled="busy" @click="addStaff">创建员工账号</button
            ><text v-if="staffError" class="error">{{ staffError }}</text></view
          ><view v-if="can('product:manage')" class="panel"
            ><view class="section-row"><text class="section-title">商品与套餐</text><button class="text-button" :disabled="productLoading" @click="loadProducts">{{ productLoading ? "读取中…" : "刷新" }}</button></view
            ><view v-for="product in managedProducts" :key="product.productId" class="staff-row product-row"><view><b>{{ product.name }}</b><text>¥{{ money(product.price) }} · {{ product.active ? "已上架" : "已下架" }}</text></view><view class="staff-actions"><button :disabled="busy" @click="editProductName(product)">改名</button><button :disabled="busy || !product.active" @click="editProductPrice(product)">改价</button><button :class="{ danger: product.active }" :disabled="busy" @click="toggleProduct(product)">{{ product.active ? "下架" : "上架" }}</button></view></view
            ><text class="field-label">新增商品</text><label>商品名称<input v-model="productForm.name" maxlength="40" placeholder="例如：威士忌套餐" /></label><label>价格（元）<input v-model="productForm.price" type="number" placeholder="请输入整数金额" /></label><button class="primary full-button" :disabled="busy" @click="addProduct">创建并上架</button><text v-if="productError" class="error">{{ productError }}</text></view
          ><text class="footnote"
            >操作权限由服务端强制校验，操作记录会保存员工姓名。请勿录入真实客户资料。</text
          ><button :disabled="busy" @click="signOut">退出登录</button></template
        >
      </template>
      <view class="tabbar"
        ><button
          v-for="(label, key) in titles"
          :key="key"
          :class="{ active: tab === key }"
          @click="selectTab(key)"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
          >
            <path :d="icons[key]" /></svg
          ><text>{{ label }}</text>
        </button></view
      >
    </template>
    <view v-if="selectedTable" class="overlay"
      ><view class="overlay-mask" @click="selectedTable = null"></view
      ><view class="sheet"
        ><view class="sheet-heading"
          ><text>{{ selectedTable.id }} · {{ selectedTable.zone }}</text
          ><button @click="selectedTable = null">✕</button></view
        ><text class="muted"
          >建议 {{ selectedTable.capacity }} 人 · {{ tableDate }}</text
        ><text :class="['badge', tableStatus(selectedTable.id)]">{{
          labels[tableStatus(selectedTable.id)]
        }}</text
        ><template v-if="tableOrder(selectedTable.id)"
          ><view class="info-row"
            ><text>客户</text
            ><b>{{ tableOrder(selectedTable.id)?.customerName }}</b></view
          ><button
            class="primary"
            @click="
              selected = tableOrder(selectedTable.id) || null;
              selectedTable = null;
            "
          >
            查看订单与操作
          </button></template
        ><button
          v-else-if="can('order:create')"
          class="primary"
          @click="startBooking(selectedTable.id)"
        >
          为此台创建预订</button
        ><text v-else class="footnote">当前角色只能查看空闲台位。</text></view
      ></view
    >
    <view v-if="booking" class="overlay"
      ><view class="overlay-mask" @click="!busy && (booking = false)"></view
      ><view class="sheet tall"
        ><view class="sheet-heading"
          ><text>创建预订</text
          ><button :disabled="busy" @click="booking = false">✕</button></view
        ><view class="stepper"
          ><text :class="{ current: step === 1 }">01 客户与时间</text
          ><text>────</text
          ><text :class="{ current: step === 2 }">02 台位与套餐</text></view
        ><template v-if="step === 1"
          ><label
            >营业日期<picker
              mode="date"
              :value="form.date"
              :start="state?.businessDate"
              @change="
                form.date = $event.detail.value;
                form.tableId = '';
              "
              ><view class="field">{{ form.date }} ▾</view></picker
            ></label
          ><label
            >预计到店<picker
              mode="time"
              :value="form.arrivalTime"
              @change="form.arrivalTime = $event.detail.value"
              ><view class="field">{{ form.arrivalTime }} ▾</view></picker
            ></label
          ><label>预计使用时长<picker :range="durationChoices.map((d) => d.name)" @change="form.durationMinutes = durationChoices[Number($event.detail.value)].minutes; form.tableId = ''"><view class="field">{{ durationChoices.find((d) => d.minutes === form.durationMinutes)?.name }} ▾</view></picker></label
          ><label
            >到店人数<input v-model="form.guestCount" type="number" /></label
          ><label
            >客户姓名<input
              v-model="form.customerName"
              maxlength="40"
              placeholder="请输入测试客户姓名" /></label
          ><label
            >手机号码<input
              v-model="form.phone"
              type="number"
              maxlength="11"
              placeholder="11 位测试手机号" /></label
          ><label
            >订单备注<input
              v-model="form.remark"
              maxlength="200"
              placeholder="生日、忌口或其他需求（选填）" /></label
          ><button class="primary" @click="nextStep">
            下一步 · 选择台位 →
          </button></template
        ><template v-else
          ><text class="muted"
            >{{ form.customerName }} · {{ form.guestCount }} 人 ·
            {{ form.date }} {{ form.arrivalTime }}</text
          ><text class="field-label">选择可用台位</text
          ><view class="seats"
            ><button
              v-for="t in availableTables"
              :key="t.id"
              :class="['seat', { picked: form.tableId === t.id }]"
              @click="form.tableId = t.id"
            >
              <b>{{ t.id }}</b
              ><text>{{ t.capacity }} 人 · {{ t.zone }}</text>
            </button></view
          ><text v-if="!availableTables.length" class="error"
            >暂无符合人数的可用台位，请返回调整。</text
          ><text class="field-label">套餐选择</text
          ><button
            :class="['package', { picked: !packageID }]"
            @click="packageID = ''"
          >
            到店后选购 <b>¥0</b></button
          ><button
            v-for="p in state?.products.slice(0, 2)"
            :key="p.productId"
            :class="['package', { picked: packageID === p.productId }]"
            @click="packageID = p.productId"
          >
            {{ p.name }} <b>¥{{ money(p.price) }}</b></button
          ><view class="sticky-actions"
            ><button :disabled="busy" @click="step = 1">上一步</button
            ><button class="primary" :disabled="busy" @click="submitBooking">
              {{ busy ? "创建中…" : "确认预订" }}
            </button></view
          ></template
        ><text v-if="formError" class="error">{{ formError }}</text></view
      ></view
    >
    <view v-if="selected" class="overlay"
      ><view class="overlay-mask" @click="closeOrder"></view
      ><view class="sheet tall"
        ><view class="sheet-heading"
          ><text>{{ selected.tableId }} · {{ selected.customerName }}</text
          ><button id="close-order" @click="closeOrder">✕</button></view
        ><text :class="['badge', selected.status]">{{
          labels[selected.status]
        }}</text
        ><view class="info-row"
          ><text>到店时间</text
          ><b>{{ selected.date }} {{ selected.arrivalTime }}</b></view
        ><view class="info-row"><text>预计时长</text><b>{{ selected.durationMinutes / 60 }} 小时</b></view
        ><view class="info-row"
          ><text>联系电话</text
          ><b>{{
            selected.phone.replace(/(\d{3})\d{4}(\d{4})/, "$1****$2")
          }}</b></view
        ><view class="info-row"
          ><text>到店人数</text><b>{{ selected.guestCount }} 人</b></view
        ><view v-if="selected.remark" class="info-row"
          ><text>订单备注</text><b>{{ selected.remark }}</b></view
        ><view v-if="selected.cancelReason" class="info-row"
          ><text>取消原因</text><b>{{ selected.cancelReason }}</b></view
        ><button v-if="selected.status === 'reserved' && canUpdate(selected) && !editingBooking" class="inline-action" @click="startEditBooking">编辑预订资料</button
        ><view v-if="editingBooking" class="edit-booking"><text class="field-label">编辑预订资料</text><label>客户姓名<input v-model="editForm.customerName" maxlength="40" /></label><label>手机号码<input v-model="editForm.phone" type="number" maxlength="11" /></label><label>到店人数<input v-model="editForm.guestCount" type="number" /></label><label>营业日期<picker mode="date" :value="editForm.date" :start="state?.businessDate" @change="editForm.date = $event.detail.value"><view class="field">{{ editForm.date }} ▾</view></picker></label><label>预计到店<picker mode="time" :value="editForm.arrivalTime" @change="editForm.arrivalTime = $event.detail.value"><view class="field">{{ editForm.arrivalTime }} ▾</view></picker></label><label>预计时长<picker :range="durationChoices.map((d) => d.name)" @change="editForm.durationMinutes = durationChoices[Number($event.detail.value)].minutes"><view class="field">{{ durationChoices.find((d) => d.minutes === editForm.durationMinutes)?.name }} ▾</view></picker></label><label>订单备注<input v-model="editForm.remark" maxlength="200" /></label><view class="edit-actions"><button :disabled="busy" @click="editingBooking = false">取消编辑</button><button class="primary" :disabled="busy" @click="saveBookingEdit">保存修改</button></view></view
        ><template
          v-if="
            ['reserved', 'arrived'].includes(selected.status) &&
            can('order:change-table')
          "
          ><text class="field-label">调整台位</text
          ><picker
            :range="
              changeTableOptions.map(
                (t) => `${t.id} · ${t.zone} · ${t.capacity}人`,
              )
            "
            @change="
              changeTableID =
                changeTableOptions[Number($event.detail.value)]?.id || ''
            "
            ><view class="field"
              >{{ changeTableID || "选择新的空闲台位" }} ▾</view
            ></picker
          ><button
            class="inline-action"
            :disabled="busy || !changeTableID"
            @click="
              confirmAction(
                'change-table',
                `确认将 ${selected.tableId} 调整为 ${changeTableID}？`,
                { tableId: changeTableID },
              )
            "
          >
            确认改台
          </button></template
        ><text class="field-label">消费明细</text
        ><view v-if="!selected.items.length" class="muted">尚未添加商品</view
        ><view v-for="(p, i) in selected.items" :key="i" class="info-row"
          ><text>{{ p.name }} × {{ p.qty }}</text
          ><b>¥{{ money(p.price * p.qty) }}</b></view
        ><view class="info-row"
          ><text>应收金额</text
          ><b class="accent">¥{{ money(total(selected)) }}</b></view
        ><view v-if="selected.paymentMethod" class="info-row"
          ><text>已收 · {{ selected.paymentMethod }}</text
          ><b>¥{{ money(selected.paidAmount) }}</b></view
        ><view v-if="selected.refundedAmount" class="refund-record"><view class="info-row"><text>累计已退款</text><b>−¥{{ money(selected.refundedAmount) }}</b></view><template v-if="selected.refunds?.length"><text v-for="(refund, index) in selected.refunds" :key="index">¥{{ money(refund.amount) }} · {{ refund.reason }} · {{ refund.operator }} · {{ new Date(refund.at).toLocaleString('zh-CN', { hour12: false }) }}</text></template><text v-else>{{ selected.refundReason }} · {{ new Date(selected.refundedAt || '').toLocaleString('zh-CN', { hour12: false }) }}</text></view
        ><view v-if="refundEditing" class="edit-booking refund-form"><text class="field-label">登记线下退款</text><label>退款金额（元）<input v-model="refundForm.amount" type="number" /></label><label>退款原因<picker :range="refundReasons" @change="refundForm.reason = refundReasons[Number($event.detail.value)]"><view class="field">{{ refundForm.reason }} ▾</view></picker></label><text class="footnote">请先在线下完成退款，剩余可登记 ¥{{ money(refundRemaining) }}</text><view class="edit-actions"><button @click="refundEditing = false">取消</button><button class="primary" :disabled="busy" @click="submitRefund">确认登记</button></view></view
        ><template v-if="selected.status === 'serving' && can('order:add-item')"
          ><text class="field-label">添加商品</text
          ><picker
            :range="state?.products.map((p) => `${p.name} · ¥${p.price}`)"
            :value="productIndex"
            @change="productIndex = Number($event.detail.value)"
            ><view class="field"
              >{{ state?.products[productIndex]?.name }} ▾</view
            ></picker
          ><view class="quantity"
            ><button @click="quantity = Math.max(1, quantity - 1)">−</button
            ><text>{{ quantity }}</text
            ><button @click="quantity = Math.min(99, quantity + 1)">＋</button
            ><button
              :disabled="busy"
              @click="
                operate('items', {
                  items: [
                    {
                      productId: state?.products[productIndex]?.productId,
                      qty: quantity,
                    },
                  ],
                })
              "
            >
              确认加单
            </button></view
          ><text class="field-label">线下收款</text
          ><view class="chips"
            ><button
              v-for="p in ['微信', '支付宝', '现金']"
              :key="p"
              :class="{ chosen: payment === p }"
              @click="payment = p"
            >
              {{ p }}
            </button></view
          ></template
        ><text class="field-label">操作记录</text
        ><view v-for="(e, i) in selected.events" :key="i" class="event"
          ><text
            >{{ e.action
            }}<small v-if="e.operator"> · {{ e.operator }}</small></text
          ><text class="muted">{{
            new Date(e.at).toLocaleString("zh-CN", { hour12: false })
          }}</text></view
        ><text class="footnote">订单号 {{ selected.id }}</text
        ><text v-if="actionError" class="error">{{ actionError }}</text
        ><view class="sticky-actions"
          ><template v-if="selected.status === 'reserved'"
            ><button
              v-if="canCancel(selected)"
              class="danger"
              :disabled="busy"
              @click="cancelOrder"
            >
              取消预订</button
            ><button
              v-if="can('order:confirm-arrival')"
              class="primary"
              :disabled="busy"
              @click="operate('confirm-arrival')"
            >
              确认到店
            </button></template
          ><button
            v-if="selected.status === 'arrived' && can('table:open')"
            class="primary"
            :disabled="busy"
            @click="operate('open-table')"
          >
            开台 · 开始服务</button
          ><button
            v-if="selected.status === 'serving' && can('payment:checkout')"
            class="primary"
            :disabled="busy"
            @click="
              confirmAction(
                'checkout',
                `确认已通过${payment}收到 ¥${money(total(selected))}？`,
                { paymentMethod: payment },
              )
            "
          >
            确认已收款 ¥{{ money(total(selected)) }}</button
          ><button
            v-if="selected.status === 'cleaning' && can('table:clean')"
            class="primary"
            :disabled="busy"
            @click="operate('complete-cleaning')"
          >
            完成清台 · 释放台位
          </button><button v-if="['cleaning', 'completed'].includes(selected.status) && selected.paidAmount > (selected.refundedAmount || 0) && can('payment:refund') && !refundEditing" class="danger" :disabled="busy" @click="refundOrder">登记退款</button></view
        ></view
      ></view
    >
  </view>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import {
  ApiError,
  request,
  login,
  logout,
  session,
  requestKey,
  total,
  type State,
  type Order,
  type Table,
  type AuditLog,
  type StaffAccount,
  type DailyReport,
  type Product,
} from "../services/api";
const titles = { home: "工作台", tables: "台位", orders: "订单", mine: "我的" },
  icons = {
    home: "M3 10 12 3l9 7v11h-6v-7H9v7H3Z",
    tables: "M3 3h7v7H3ZM14 3h7v7h-7ZM3 14h7v7H3ZM14 14h7v7h-7Z",
    orders: "M5 3h14v18H5ZM8 8h8M8 12h8M8 16h5",
    mine: "M8 7a4 4 0 1 0 8 0 4 4 0 1 0-8 0M4 21v-3a8 6 0 0 1 16 0v3",
  };
const labels: Record<string, string> = {
  free: "空闲",
  reserved: "待到店",
  arrived: "已到店",
  serving: "消费中",
  cleaning: "待清台",
  completed: "已完成",
  cancelled: "已取消",
};
const tab = ref<keyof typeof titles>("home"),
  authenticated = ref(!!session()),
  code = ref(""),
  selectedStaff = ref("owner"),
  busy = ref(false),
  loading = ref(false),
  error = ref(""),
  headerInset = ref({ top: 0, right: 0 }),
  keyboardInset = ref(0),
  appVisible = ref(true),
  state = ref<State | null>(null),
  selected = ref<Order | null>(null),
  selectedTable = ref<Table | null>(null),
  query = ref(""),
  orderFilter = ref("all"),
  orderResults = ref<Order[]>([]),
  orderCursor = ref(""),
  orderLoading = ref(false),
  orderListError = ref(""),
  tableFilter = ref("all"),
  tableDate = ref(""),
  actionError = ref("");
const auditLogs = ref<AuditLog[]>([]),
  auditLoading = ref(false),
  auditError = ref(""),
  auditFrom = ref(""),
  auditTo = ref(""),
  auditUser = ref(""),
  auditCategory = ref("all");
const auditCategories = [{ id: "all", name: "全部操作" }, { id: "payment", name: "收款与退款" }, { id: "refund", name: "仅退款" }];
const staffAccounts = ref<StaffAccount[]>([]),
  staffLoading = ref(false),
  staffError = ref(""),
  staffForm = ref({ name: "", role: "sales", pin: "" });
const report = ref<DailyReport | null>(null),
  reportLoading = ref(false),
  reportError = ref(""),
  reportFrom = ref(""),
  reportTo = ref("");
const managedProducts = ref<Product[]>([]),
  productLoading = ref(false),
  productError = ref(""),
  productForm = ref({ name: "", price: "" });
const staffChoices = ref([
  { id: "sales", name: "销售小林", roleName: "销售" },
  { id: "frontdesk", name: "前台小周", roleName: "前台" },
  { id: "waiter", name: "服务员阿杰", roleName: "服务员" },
  { id: "owner", name: "店长", roleName: "店长" },
]);
const roleChoices = [
  { id: "sales", name: "销售" },
  { id: "frontdesk", name: "前台" },
  { id: "waiter", name: "服务员" },
  { id: "owner", name: "店长" },
];
function can(permission: string) {
  const permissions = state.value?.currentUser.permissions || [];
  return permissions.includes("*") || permissions.includes(permission);
}
function shanghaiDate(offsetDays = 0) {
  const value = new Date(Date.now() + offsetDays * 86400000);
  const parts = new Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Shanghai", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(value);
  const part = (type: string) => parts.find((item) => item.type === type)?.value || "";
  return `${part("year")}-${part("month")}-${part("day")}`;
}
function canCancel(order: Order) {
  return (
    can("order:cancel") ||
    (can("order:cancel-own") && order.createdBy === state.value?.currentUser.id)
  );
}
function canUpdate(order: Order) {
  return can("order:update") || (can("order:update-own") && order.createdBy === state.value?.currentUser.id);
}
const canReport = computed(() => can("report:view") || can("report:own"));
const permissionSummary = computed(() => {
  if (can("*")) return "全部运营权限";
  const names: Record<string, string> = {
    "order:create": "创建预订",
    "order:update": "修改预订",
    "order:update-own": "修改本人预订",
    "order:confirm-arrival": "确认到店",
    "table:open": "开台",
    "order:add-item": "加单",
    "payment:checkout": "结账",
    "table:clean": "清台",
    "order:cancel": "取消预订",
    "order:cancel-own": "取消本人预订",
    "order:change-table": "改台",
    "report:own": "本人业绩",
  };
  return (
    (state.value?.currentUser.permissions || [])
      .map((p) => names[p] || p)
      .join("、") || "只读"
  );
});
const tableFilters = [
    { key: "all", label: "全部" },
    { key: "free", label: "空闲" },
    { key: "reserved", label: "预订" },
    { key: "serving", label: "消费中" },
    { key: "cleaning", label: "待清台" },
  ],
  orderFilters = [
    { key: "all", label: "全部" },
    { key: "reserved", label: "待到店" },
    { key: "active", label: "进行中" },
    { key: "completed", label: "已完成" },
    { key: "cancelled", label: "已取消" },
  ];
const money = (n: number) => n.toLocaleString("zh-CN"),
  todayOrders = computed(
    () =>
      state.value?.orders.filter((o) => o.date === state.value?.businessDate) ||
      [],
  ),
  revenue = computed(() =>
    todayOrders.value
      .filter((o) => ["cleaning", "completed"].includes(o.status))
      .reduce((s, o) => s + o.paidAmount - (o.refundedAmount || 0), 0),
  ),
  pending = computed(() =>
    todayOrders.value.filter((o) =>
      ["reserved", "cleaning"].includes(o.status),
    ),
  ),
  updatedTime = computed(() =>
    state.value
      ? new Date(state.value.updatedAt).toLocaleTimeString("zh-CN", {
          hour12: false,
        })
      : "—",
  );
function tableOrder(id: string) {
  return state.value?.orders.find(
    (o) =>
      o.tableId === id &&
      o.date === tableDate.value &&
      !["completed", "cancelled"].includes(o.status),
  );
}
function tableStatus(id: string) {
  return tableOrder(id)?.status || "free";
}
function tableCount(key: string) {
  return (
    state.value?.tables.filter(
      (t) => key === "all" || tableStatus(t.id) === key,
    ).length || 0
  );
}
const visibleTables = computed(
    () =>
      state.value?.tables.filter(
        (t) =>
          tableFilter.value === "all" ||
          tableStatus(t.id) === tableFilter.value,
      ) || [],
  ),
  filteredOrders = computed(() => orderResults.value);
let refreshInFlight = false;
async function refresh(mode: "full" | "poll" = "full") {
  if (!authenticated.value || refreshInFlight) return;
  refreshInFlight = true;
  if (mode === "full") loading.value = true;
  try {
    const path =
      mode === "poll" && state.value?.rev
        ? `/state?rev=${state.value.rev}`
        : "/state";
    let incoming = await request<State>(path);
    if (
      incoming.unchanged &&
      incoming.businessDate &&
      incoming.businessDate !== state.value?.businessDate
    ) {
      incoming = await request<State>("/state");
    }
    if (incoming.unchanged) {
      error.value = "";
      return;
    }
    // An older in-flight poll must not overwrite a write response received later.
    const known = new Map((state.value?.orders || []).map((o) => [o.id, o]));
    if (selected.value) known.set(selected.value.id, selected.value);
    incoming.orders = incoming.orders.map((o) => {
      const cached = known.get(o.id);
      return cached && cached.version > o.version ? cached : o;
    });
    state.value = incoming;
    error.value = "";
    if (!tableDate.value) tableDate.value = state.value.businessDate;
    if (!reportTo.value) {
      reportTo.value = state.value.businessDate;
      const start = new Date(`${state.value.businessDate}T12:00:00+08:00`);
      start.setDate(start.getDate() - 6);
      reportFrom.value = start.toISOString().slice(0, 10);
    }
    if (!auditTo.value) {
      auditFrom.value = shanghaiDate(-6);
      auditTo.value = shanghaiDate();
    }
    if (selected.value)
      selected.value =
        state.value.orders.find((o) => o.id === selected.value?.id) || null;
  } catch (e) {
    error.value = (e as Error).message;
    if (!session()) authenticated.value = false;
  } finally {
    refreshInFlight = false;
    loading.value = false;
  }
}
function closeOrder() {
  selected.value = null;
  actionError.value = "";
}
async function signIn() {
  busy.value = true;
  error.value = "";
  try {
    await login(code.value, selectedStaff.value);
    authenticated.value = true;
    code.value = "";
    await refresh();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
function dropSession() {
  authenticated.value = false;
  state.value = null;
  selected.value = null;
  auditLogs.value = [];
  staffAccounts.value = [];
  managedProducts.value = [];
  orderResults.value = [];
  orderCursor.value = "";
  report.value = null;
}
function openPrivacy() {
  uni.navigateTo({ url: "/pages/privacy/index" });
}
function signOut() {
  busy.value = true;
  logout().finally(() => {
    dropSession();
    loadLoginOptions();
    busy.value = false;
  });
}
async function loadLoginOptions() {
  try {
    const result = await request<{ staff: { id: string; name: string; roleName: string }[] }>("/login-options");
    if (result.staff.length) {
      staffChoices.value = result.staff;
      if (!result.staff.some((person) => person.id === selectedStaff.value))
        selectedStaff.value = result.staff[0].id;
    }
  } catch {
    // Keep the migration defaults visible if the public options request is unavailable.
  }
}
async function loadAudit() {
  if (!can("audit:view") || auditLoading.value) return;
  auditLoading.value = true;
  auditError.value = "";
  try {
    const params = new URLSearchParams({ from: auditFrom.value, to: auditTo.value, userId: auditUser.value, category: auditCategory.value });
    const result = await request<{ logs: AuditLog[] }>(`/audit-logs?${params.toString()}`);
    auditLogs.value = result.logs;
  } catch (e) {
    auditError.value = (e as Error).message;
  } finally {
    auditLoading.value = false;
  }
}
function csvCell(value: unknown) {
  let text = String(value ?? "");
  if (/^[=+\-@]/.test(text)) text = `'${text}`;
  return `"${text.replace(/"/g, '""')}"`;
}
function downloadCSV(filename: string, rows: unknown[][]) {
  if (typeof document === "undefined") {
    uni.showToast({ title: "请在网页端导出", icon: "none" });
    return;
  }
  const blob = new Blob(["\uFEFF" + rows.map((row) => row.map(csvCell).join(",")).join("\r\n")], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob), link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
function exportReport() {
  if (!report.value) return;
  downloadCSV(`HOLE-CLUB-营业报表-${report.value.from}-${report.value.to}.csv`, [
    ["营业日期", "预订数", "已结账订单", "取消订单", "到店人数", "净营业收入", "退款笔数", "退款金额"],
    ...report.value.days.map((day) => [day.date, day.reservations, day.completed, day.cancelled, day.guests, day.revenue, day.refunded, day.refundAmount]),
    ["合计", report.value.totals.reservations, report.value.totals.completed, report.value.totals.cancelled, report.value.totals.guests, report.value.totals.revenue, report.value.totals.refunded, report.value.totals.refundAmount],
  ]);
}
function exportAudit() {
  downloadCSV(`HOLE-CLUB-操作审计-${auditFrom.value}-${auditTo.value}.csv`, [
    ["操作时间", "员工", "角色", "操作", "订单号"],
    ...auditLogs.value.map((log) => [new Date(log.createdAt).toLocaleString("zh-CN", { hour12: false }), log.userName, log.roleName, log.action, log.orderId]),
  ]);
}
async function loadReport() {
  if (!canReport.value || reportLoading.value || !reportFrom.value || !reportTo.value) return;
  reportLoading.value = true;
  reportError.value = "";
  try {
    report.value = await request<DailyReport>(`/reports/daily?from=${encodeURIComponent(reportFrom.value)}&to=${encodeURIComponent(reportTo.value)}`);
  } catch (e) {
    reportError.value = (e as Error).message;
  } finally {
    reportLoading.value = false;
  }
}
async function loadStaff() {
  if (!can("staff:manage") || staffLoading.value) return;
  staffLoading.value = true;
  staffError.value = "";
  try {
    const result = await request<{ staff: StaffAccount[] }>("/staff");
    staffAccounts.value = result.staff;
  } catch (e) {
    staffError.value = (e as Error).message;
  } finally {
    staffLoading.value = false;
  }
}
async function loadProducts() {
  if (!can("product:manage") || productLoading.value) return;
  productLoading.value = true;
  productError.value = "";
  try {
    const result = await request<{ products: Product[] }>("/products");
    managedProducts.value = result.products;
  } catch (e) {
    productError.value = (e as Error).message;
  } finally {
    productLoading.value = false;
  }
}
async function addProduct() {
  const price = Number(productForm.value.price);
  productError.value = "";
  if (!productForm.value.name.trim() || !Number.isInteger(price) || price < 1) {
    productError.value = "请填写商品名称和大于 0 的整数价格";
    return;
  }
  busy.value = true;
  try {
    await request("/products", "POST", { name: productForm.value.name, price });
    productForm.value = { name: "", price: "" };
    await Promise.all([loadProducts(), refresh()]);
    uni.showToast({ title: "商品已上架", icon: "success" });
  } catch (e) {
    productError.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function updateProduct(product: Product, changes: Record<string, unknown>, message: string) {
  busy.value = true;
  productError.value = "";
  try {
    await request(`/products/${product.productId}`, "PUT", changes);
    await Promise.all([loadProducts(), refresh()]);
    uni.showToast({ title: message, icon: "success" });
  } catch (e) {
    productError.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
function editProductName(product: Product) {
  uni.showModal({ title: "修改商品名称", content: product.name, editable: true, placeholderText: "输入新名称", success: (choice) => {
    if (!choice.confirm) return;
    const name = (choice.content || "").trim();
    if (!name) productError.value = "商品名称不能为空";
    else updateProduct(product, { name }, "名称已更新");
  } });
}
function editProductPrice(product: Product) {
  uni.showModal({ title: `修改 ${product.name} 价格`, content: String(product.price), editable: true, placeholderText: "输入整数金额", success: (choice) => {
    if (!choice.confirm) return;
    const price = Number((choice.content || "").trim());
    if (!Number.isInteger(price) || price < 1) productError.value = "价格需为大于 0 的整数";
    else updateProduct(product, { price }, "价格已更新");
  } });
}
function toggleProduct(product: Product) {
  uni.showModal({ title: product.active ? "下架商品" : "上架商品", content: `${product.name}\n${product.active ? "下架后不能再加入新订单，历史账单不受影响。" : "上架后可立即用于开单和加单。"}`, success: (choice) => {
    if (choice.confirm) updateProduct(product, { active: !product.active }, product.active ? "商品已下架" : "商品已上架");
  } });
}
async function addStaff() {
  staffError.value = "";
  if (!staffForm.value.name.trim() || staffForm.value.pin.length < 6) {
    staffError.value = "请填写员工姓名和至少 6 位初始口令";
    return;
  }
  busy.value = true;
  try {
    await request("/staff", "POST", staffForm.value);
    staffForm.value = { name: "", role: "sales", pin: "" };
    await Promise.all([loadStaff(), loadLoginOptions()]);
    uni.showToast({ title: "员工已创建", icon: "success" });
  } catch (e) {
    staffError.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
function toggleStaff(person: StaffAccount) {
  uni.showModal({
    title: person.active ? "停用员工" : "启用员工",
    content: `${person.active ? "停用后该员工会立即退出登录。" : "确认恢复该员工登录权限？"}\n${person.name} · ${person.roleName}`,
    success: async (choice) => {
      if (!choice.confirm) return;
      busy.value = true;
      staffError.value = "";
      try {
        await request(`/staff/${person.id}`, "PUT", { active: !person.active });
        await Promise.all([loadStaff(), loadLoginOptions()]);
      } catch (e) {
        staffError.value = (e as Error).message;
      } finally {
        busy.value = false;
      }
    },
  });
}
function resetStaffPin(person: StaffAccount) {
  uni.showModal({
    title: `重置 ${person.name} 的口令`,
    content: "",
    editable: true,
    placeholderText: "输入 6–32 位新口令",
    success: async (choice) => {
      if (!choice.confirm) return;
      const pin = (choice.content || "").trim();
      if (pin.length < 6 || pin.length > 32) {
        staffError.value = "新口令需为 6–32 个字符";
        return;
      }
      busy.value = true;
      staffError.value = "";
      try {
        await request(`/staff/${person.id}`, "PUT", { pin });
        if (person.id === state.value?.currentUser.id) {
          uni.showToast({ title: "口令已更新，请重新登录", icon: "none" });
          signOut();
        } else {
          uni.showToast({ title: "口令已重置", icon: "success" });
        }
      } catch (e) {
        staffError.value = (e as Error).message;
      } finally {
        busy.value = false;
      }
    },
  });
}
watch([tab, () => state.value?.currentUser.id], ([value]) => {
  if (value === "mine" && state.value) {
    if (canReport.value) loadReport();
    if (can("audit:view")) loadAudit();
    if (can("staff:manage")) loadStaff();
    if (can("product:manage")) loadProducts();
  }
});
function selectTab(value: keyof typeof titles) {
  tab.value = value;
  if (value === "orders") loadOrders(true);
  if (value === "mine") {
    // Trigger explicitly as well as through the watcher so the first navigation
    // after login cannot race the state response that supplies permissions.
    if (can("audit:view")) loadAudit();
    if (can("staff:manage")) loadStaff();
    if (can("product:manage")) loadProducts();
    if (canReport.value) loadReport();
  }
}
function showOrders(filter: string) {
  orderFilter.value = filter;
  tab.value = "orders";
  loadOrders(true);
}
let orderRequestSequence = 0;
async function loadOrders(reset: boolean) {
  if (!authenticated.value || (!reset && (!orderCursor.value || orderLoading.value))) return;
  const sequence = ++orderRequestSequence;
  orderLoading.value = true;
  orderListError.value = "";
  try {
    const params = new URLSearchParams({ limit: "20", status: orderFilter.value, q: query.value.trim() });
    if (!reset && orderCursor.value) params.set("cursor", orderCursor.value);
    const result = await request<{ orders: Order[]; nextCursor: string }>(`/orders?${params.toString()}`);
    if (sequence !== orderRequestSequence) return;
    orderResults.value = reset ? result.orders : [...orderResults.value, ...result.orders];
    orderCursor.value = result.nextCursor;
  } catch (e) {
    if (sequence === orderRequestSequence) orderListError.value = (e as Error).message;
  } finally {
    if (sequence === orderRequestSequence) orderLoading.value = false;
  }
}
let orderSearchTimer: ReturnType<typeof setTimeout>;
watch([orderFilter, query], () => {
  if (tab.value !== "orders") return;
  clearTimeout(orderSearchTimer);
  orderSearchTimer = setTimeout(() => loadOrders(true), 300);
});
const booking = ref(false),
  step = ref(1),
  formError = ref(""),
  packageID = ref(""),
  bookingKey = ref(""),
  form = ref({
    customerName: "",
    phone: "",
    guestCount: 6,
    date: "",
    arrivalTime: "22:30",
    durationMinutes: 240,
    tableId: "",
    remark: "",
  });
const durationChoices = [
  { name: "2 小时", minutes: 120 },
  { name: "4 小时", minutes: 240 },
  { name: "6 小时", minutes: 360 },
  { name: "8 小时", minutes: 480 },
];
function minuteOf(value: string) {
  const [hour, minute] = value.split(":").map(Number);
  return hour * 60 + minute;
}
function overlaps(order: Order, date: string, arrival: string, duration: number) {
  if (order.date !== date || ["completed", "cancelled"].includes(order.status)) return false;
  const left = minuteOf(arrival), right = left + duration;
  const orderLeft = minuteOf(order.arrivalTime), orderRight = orderLeft + (order.durationMinutes || 240);
  return orderLeft < right && orderRight > left;
}
const availableTables = computed(
  () =>
    state.value?.tables.filter(
      (t) =>
        t.capacity >= Number(form.value.guestCount) &&
        !state.value?.orders.some(
          (o) =>
            o.tableId === t.id &&
            overlaps(o, form.value.date, form.value.arrivalTime, form.value.durationMinutes),
        ),
    ) || [],
);
const changeTableID = ref("");
const editingBooking = ref(false),
  editForm = ref({ customerName: "", phone: "", guestCount: 1, date: "", arrivalTime: "", durationMinutes: 240, remark: "" });
function startEditBooking() {
  if (!selected.value) return;
  editForm.value = {
    customerName: selected.value.customerName,
    phone: selected.value.phone,
    guestCount: selected.value.guestCount,
    date: selected.value.date,
    arrivalTime: selected.value.arrivalTime,
    durationMinutes: selected.value.durationMinutes || 240,
    remark: selected.value.remark || "",
  };
  editingBooking.value = true;
}
function saveBookingEdit() {
  const data = { ...editForm.value, guestCount: Number(editForm.value.guestCount) };
  if (!data.customerName.trim() || !/^1\d{10}$/.test(data.phone) || !Number.isInteger(data.guestCount) || data.guestCount < 1) {
    actionError.value = "请检查客户姓名、11 位手机号和到店人数";
    return;
  }
  operate("update-booking", data);
}
const changeTableOptions = computed(() =>
  (state.value?.tables || []).filter(
    (t) =>
      t.id !== selected.value?.tableId &&
      t.capacity >= (selected.value?.guestCount || 1) &&
      !state.value?.orders.some(
        (o) =>
      o.id !== selected.value?.id &&
          o.tableId === t.id &&
          !!selected.value && overlaps(o, selected.value.date, selected.value.arrivalTime, selected.value.durationMinutes || 240),
      ),
  ),
);
function startBooking(id = "") {
  form.value = {
    customerName: "",
    phone: "",
    guestCount: 6,
    date: tableDate.value || state.value?.businessDate || "",
    arrivalTime: "22:30",
    durationMinutes: 240,
    tableId: id,
    remark: "",
  };
  step.value = 1;
  formError.value = "";
  packageID.value = "";
  bookingKey.value = requestKey();
  booking.value = true;
  selectedTable.value = null;
}
function nextStep() {
  formError.value = "";
  if (!form.value.customerName.trim() || !/^1\d{10}$/.test(form.value.phone)) {
    formError.value = "请填写客户姓名和 11 位手机号";
    return;
  }
  if (
    !Number.isInteger(Number(form.value.guestCount)) ||
    Number(form.value.guestCount) < 1 ||
    Number(form.value.guestCount) > 10
  ) {
    formError.value = "人数需为 1–10 的整数";
    return;
  }
  step.value = 2;
}
watch(
  [form, packageID],
  () => {
    bookingKey.value = requestKey();
  },
  { deep: true },
);
async function submitBooking() {
  if (busy.value) return;
  if (!availableTables.value.some((t) => t.id === form.value.tableId)) {
    formError.value = "请选择一个可用台位";
    return;
  }
  busy.value = true;
  formError.value = "";
  try {
    const r = await request<{ order: Order }>(
      "/orders",
      "POST",
      {
        ...form.value,
        guestCount: Number(form.value.guestCount),
        items: packageID.value ? [{ productId: packageID.value, qty: 1 }] : [],
      },
      bookingKey.value,
    );
    booking.value = false;
    await refresh();
    selected.value = r.order;
    tab.value = "orders";
    loadOrders(true);
    uni.showToast({ title: "预订创建成功", icon: "success" });
  } catch (e) {
    formError.value = (e as Error).message;
    await refresh();
  } finally {
    busy.value = false;
  }
}
const productIndex = ref(2),
  quantity = ref(1),
  payment = ref("微信");
let mutation: {
  finger: string;
  key: string;
  body: Record<string, unknown>;
} | null = null;
watch(
  () => selected.value?.id,
  () => {
    actionError.value = "";
    quantity.value = 1;
    changeTableID.value = "";
    editingBooking.value = false;
    refundEditing.value = false;
    mutation = null;
  },
);
async function operate(
  action: string,
  data: Record<string, unknown> = {},
  expectedVersion?: number,
) {
  if (!selected.value || busy.value) return;
  const orderID = selected.value.id;
  busy.value = true;
  actionError.value = "";
  const path = `/orders/${selected.value.id}/${action}`,
    body = { ...data, version: expectedVersion ?? selected.value.version },
    finger = JSON.stringify({ path, data });
  if (mutation?.finger !== finger)
    mutation = { finger, key: requestKey(), body };
  try {
    const r = await request<{ order: Order }>(
      path,
      "POST",
      mutation!.body,
      mutation!.key,
    );
    if (selected.value?.id === orderID) selected.value = r.order;
    orderResults.value = orderResults.value.map((order) => order.id === r.order.id ? r.order : order);
    if (action === "change-table") changeTableID.value = "";
    if (action === "update-booking") editingBooking.value = false;
    if (action === "refund") refundEditing.value = false;
    mutation = null;
    await refresh();
    uni.showToast({ title: "操作成功", icon: "success" });
  } catch (e) {
    actionError.value = (e as Error).message;
    if (e instanceof ApiError && e.status >= 400 && e.status < 500)
      mutation = null;
    await refresh();
  } finally {
    busy.value = false;
  }
}
const cancelReasons = ["客户取消", "未按时到店", "重复预订", "其他"];
function cancelOrder() {
  uni.showActionSheet({
    itemList: cancelReasons,
    success: (choice) => {
      const reason = cancelReasons[choice.tapIndex];
      confirmAction("cancel", `确认取消预订？\n原因：${reason}`, {
        cancelReason: reason,
      });
    },
  });
}
const refundReasons = ["客户投诉退款", "重复收款", "运营异常", "其他"];
const refundEditing = ref(false),
  refundForm = ref({ amount: "", reason: refundReasons[0] });
const refundRemaining = computed(() => (selected.value?.paidAmount || 0) - (selected.value?.refundedAmount || 0));
function refundOrder() {
  refundForm.value = { amount: String(refundRemaining.value), reason: refundReasons[0] };
  refundEditing.value = true;
}
function submitRefund() {
  const refundAmount = Number(refundForm.value.amount);
  if (!Number.isInteger(refundAmount) || refundAmount < 1 || refundAmount > refundRemaining.value) {
    actionError.value = `退款金额需为 1–${money(refundRemaining.value)} 元`;
    return;
  }
  const refundReason = refundForm.value.reason;
  confirmAction("refund", `请先确认线下退款已经完成。\n本次登记退款 ¥${money(refundAmount)}\n原因：${refundReason}`, { refundReason, refundAmount });
}
function confirmAction(
  action: string,
  content: string,
  data: Record<string, unknown> = {},
) {
  const version = selected.value?.version;
  uni.showModal({
    title: "确认操作",
    content,
    confirmText: "确定",
    cancelText: "取消",
    success: (r) => {
      if (r.confirm) operate(action, data, version);
    },
  });
}
const safePad = computed(() => ({
  paddingTop: `${24 + headerInset.value.top}px`,
  paddingRight: `${18 + headerInset.value.right}px`,
  paddingBottom: `${100 + keyboardInset.value}px`,
}));
const avatarPad = computed(() =>
  headerInset.value.right ? { marginRight: `${headerInset.value.right}px` } : {},
);
function applyCapsule() {
  if (typeof document !== "undefined") return;
  try {
    const sys = uni.getSystemInfoSync();
    const status = Number(sys.statusBarHeight || 0);
    let top = status;
    let right = 0;
    const menu = uni.getMenuButtonBoundingClientRect();
    if (menu && menu.height) {
      top = menu.bottom + 8;
      right = Math.max(0, Number(sys.windowWidth || 0) - menu.left + 8);
    }
    headerInset.value = { top, right };
  } catch {
    headerInset.value = { top: 0, right: 0 };
  }
}
function onAppShown() {
  appVisible.value = true;
  if (authenticated.value) refresh("poll");
}
function onAppHidden() {
  appVisible.value = false;
}
function onKeyboard(res: { height: number }) {
  keyboardInset.value = Math.max(0, Number(res.height || 0));
}
let timer: ReturnType<typeof setInterval>;
onMounted(() => {
  uni.$on("club-session-cleared", dropSession);
  applyCapsule();
  uni.onAppShow(onAppShown);
  uni.onAppHide(onAppHidden);
  if (typeof uni.onKeyboardHeightChange === "function")
    uni.onKeyboardHeightChange(onKeyboard);
  loadLoginOptions();
  refresh();
  timer = setInterval(() => {
    if (!appVisible.value) return;
    if (
      typeof document !== "undefined" &&
      document.visibilityState !== "visible"
    )
      return;
    refresh("poll");
  }, 5000);
});
onUnmounted(() => {
  uni.$off("club-session-cleared", dropSession);
  if (typeof uni.offAppShow === "function") uni.offAppShow(onAppShown);
  if (typeof uni.offAppHide === "function") uni.offAppHide(onAppHidden);
  if (typeof uni.offKeyboardHeightChange === "function")
    uni.offKeyboardHeightChange(onKeyboard);
  clearInterval(timer);
  clearTimeout(orderSearchTimer);
});
</script>
<style scoped>
.console {
  --lime: #d7ff3f;
  --surface: #18191d;
  --line: #2a2c30;
  min-height: 100vh;
  background: #101114;
  color: #f5f3ed;
  padding: 24px 18px 100px;
  font-size: 14px;
  font-family:
    Inter,
    -apple-system,
    "PingFang SC",
    sans-serif;
}
.console text {
  line-height: 1.5;
}
.header,
.row,
.toolbar,
.section-heading,
.sheet-heading,
.connection,
.info-row,
.card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.header {
  margin-bottom: 22px;
}
.eyebrow {
  display: block;
  color: var(--lime);
  letter-spacing: 3px;
  font-size: 11px;
  font-weight: 700;
}
.page-title {
  display: block;
  font-size: 28px;
  font-weight: 700;
  margin-top: 6px;
}
.avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: #282b21 !important;
  color: var(--lime) !important;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: 1px;
}
.console button {
  margin: 0;
  background: #25272d;
  border: 1px solid transparent;
  border-radius: 10px;
  color: #f5f3ed;
  font-size: 14px;
  min-height: 44px;
  line-height: 1.4;
  padding: 12px 16px;
}
.console button::after {
  border: 0;
}
.console button[disabled] {
  opacity: 0.5;
}
.primary,
.small-primary {
  background: var(--lime) !important;
  color: #161811 !important;
  font-weight: 700;
}
.connection {
  background: #1b211a;
  border: 1px solid #303c27;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 11px;
  color: #b2c19c;
}
.muted {
  color: #9da1aa;
  font-size: 12px;
}
.welcome {
  margin: 26px 0 18px;
}
.welcome > text {
  display: block;
}
.welcome > text:last-child {
  font-size: 19px;
  margin-top: 8px;
}
.metrics {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.metric {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 17px 14px;
  min-height: 136px;
}
.metric text {
  display: block;
  font-size: 12px;
  color: #adb0b8;
}
.metric b {
  display: block;
  font-size: 36px;
  font-weight: 650;
  margin: 8px 0;
}
.metric .money {
  font-size: 26px;
  line-height: 43px;
}
.metric text:last-child {
  font-size: 11px;
}
.metric.featured {
  background: var(--lime);
  color: #15180b;
  border-color: var(--lime);
}
.metric.featured text {
  color: #465019;
}
.section-heading {
  margin: 26px 0 14px;
  font-weight: 600;
  font-size: 17px;
}
.section-heading .muted {
  font-size: 10px;
  letter-spacing: 1px;
}
.quick {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.panel {
  border: 1px solid var(--line);
  background: var(--surface);
  border-radius: 14px;
  padding: 18px;
  margin-bottom: 16px;
}
.empty,
.placeholder {
  text-align: center;
  padding: 35px 18px;
  color: #a2a7b1;
  font-size: 13px;
}
.empty button {
  margin: 16px auto 0;
}
.order-card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 12px;
  display: flex;
  align-items: center;
  padding: 14px;
  margin-bottom: 10px;
  gap: 12px;
}
.order-card.full {
  display: block;
}
.table-icon {
  background: #262d20;
  color: var(--lime);
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  flex-shrink: 0;
}
.grow {
  flex: 1;
  min-width: 0;
}
.grow b,
.grow text {
  display: block;
}
.grow b {
  font-size: 14px;
  margin-bottom: 5px;
}
.badge {
  display: inline-block;
  font-size: 11px;
  padding: 5px 9px;
  border-radius: 6px;
  background: #2a2d33;
  color: #b3b7c0;
  white-space: nowrap;
}
.badge.reserved {
  background: #372d1c;
  color: #ffc375;
}
.badge.arrived,
.badge.serving {
  background: #1f3047;
  color: #8abaff;
}
.badge.free,
.badge.completed {
  background: #213628;
  color: #88e6a7;
}
.badge.cancelled {
  background: #3b2528;
  color: #ffaaa5;
}
.card-bottom {
  padding-top: 13px;
  margin-top: 14px;
  border-top: 1px solid var(--line);
  font-size: 13px;
}
.card-bottom > text {
  color: #9299a5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 70%;
  font-size: 12px;
}
.tabbar {
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: min(480px, 100%);
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: #17181cf5;
  border-top: 1px solid var(--line);
  padding: 8px 8px calc(8px + env(safe-area-inset-bottom));
  z-index: 10;
}
/* #ifndef H5 */
.tabbar {
  left: 0;
  right: 0;
  transform: none;
  width: 100%;
}
/* #endif */
.tabbar button {
  background: none;
  color: #838894;
  font-size: 11px;
  padding: 5px;
}
.tabbar .active {
  color: var(--lime);
}
.tabbar svg {
  width: 22px;
  height: 22px;
  display: block;
  margin: 0 auto 4px;
}
.toolbar {
  margin: 16px 0;
}
.toolbar button {
  font-size: 12px;
  padding: 10px;
}
.chips {
  display: flex;
  gap: 7px;
  flex-wrap: wrap;
  margin: 12px 0;
}
.chips button {
  font-size: 11px;
  padding: 9px 11px;
  min-height: 36px;
}
.chips .chosen {
  background: var(--lime);
  color: #18200a;
}
.floor {
  padding: 16px;
}
.stage,
.bar {
  padding: 12px;
  text-align: center;
  font-size: 11px;
  letter-spacing: 2px;
  border-radius: 7px;
  background: #2b2c32;
  color: #c1c4cb;
}
.zone-heading {
  font-size: 11px;
  color: #cbcfb3;
  margin: 22px 0 12px;
  letter-spacing: 1px;
  display: flex;
  justify-content: space-between;
}
.zone-heading text {
  color: #868c96;
}
.seats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 9px;
}
.console .seat {
  min-height: 82px;
  border: 1px solid #2f5139;
  background: #1e2b22;
  color: #9ddbb0;
  padding: 10px 5px;
}
.seat b,
.seat text {
  display: block;
  font-size: 11px;
}
.seat b {
  font-size: 17px;
  margin-bottom: 5px;
}
.seat .capacity {
  font-size: 11px;
  opacity: 0.6;
}
.console .seat.reserved {
  background: #342b1c;
  border-color: #68502b;
  color: #ffc77d;
}
.console .seat.serving,
.console .seat.arrived {
  background: #1e2e43;
  border-color: #36517c;
  color: #91bdff;
}
.console .seat.cleaning {
  background: #282a30;
  border-color: #454750;
  color: #adafba;
}
.dance {
  border: 1px dashed #36383f;
  color: #666d77;
  text-align: center;
  font-size: 10px;
  letter-spacing: 2px;
  padding: 26px 0;
  margin: 20px 0 8px;
  border-radius: 8px;
}
.bar {
  margin-top: 20px;
}
.footnote {
  display: block;
  color: #838b96;
  font-size: 11px;
  margin: 18px 0;
  line-height: 1.8 !important;
}
.search,
.console input,
.field {
  background: #24262b;
  border: 1px solid #3b3d43;
  border-radius: 9px;
  color: #f3f3ed;
  height: 46px;
  padding: 0 12px;
  font-size: 14px;
  box-sizing: border-box;
}
.search {
  flex: 1;
  min-width: 0;
}
.console input:focus {
  border-color: var(--lime);
}
.field {
  display: flex;
  align-items: center;
}
.profile {
  text-align: center;
  padding: 30px;
}
.profile .avatar {
  margin: auto auto 14px;
}
.section-title {
  display: block;
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 10px;
}
.profile .muted {
  display: block;
}
.info-row {
  min-height: 47px;
  border-bottom: 1px solid var(--line);
  font-size: 13px;
}
.info-row > text {
  color: #b1b4bd;
}
.info-row b {
  font-size: 13px;
  font-weight: 500;
  text-align: right;
}
.accent {
  color: var(--lime);
}
.overlay {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.overlay-mask {
  position: absolute;
  inset: 0;
  background: #000a;
}
.sheet {
  position: relative;
  z-index: 1;
  box-sizing: border-box;
  width: min(480px, 100%);
  background: #1b1d22;
  border: 1px solid #383b43;
  border-radius: 20px 20px 0 0;
  padding: 20px 20px calc(24px + env(safe-area-inset-bottom));
  max-height: 90vh;
  overflow-y: auto;
}
.sheet.tall {
  max-height: 94vh;
}
.sheet-heading {
  font-size: 20px;
  font-weight: 650;
  margin-bottom: 18px;
}
.sheet-heading button {
  background: transparent;
  padding: 10px;
  min-width: 44px;
}
.sheet > .primary {
  width: 100%;
  margin-top: 18px;
}
.sheet > .badge {
  margin: 10px 0;
}
.stepper {
  display: flex;
  justify-content: space-between;
  color: #777e8b;
  font-size: 12px;
  margin-bottom: 24px;
}
.stepper .current {
  color: var(--lime);
}
.sheet label {
  display: block;
  color: #b5b9c1;
  font-size: 12px;
  margin: 15px 0;
}
.sheet label input,
.sheet label .field {
  margin-top: 8px;
  width: 100%;
}
.field-label {
  display: block;
  margin: 22px 0 12px;
  font-weight: 600;
  font-size: 14px;
}
.package {
  display: flex !important;
  width: 100%;
  justify-content: space-between;
  margin: 8px 0 !important;
  font-size: 13px !important;
}
.console .inline-action {
  width: 100%;
  margin-top: 9px;
  border-color: #41444d;
}
.edit-booking {
  margin-top: 14px;
  padding: 14px;
  border: 1px solid #393c44;
  border-radius: 10px;
  background: #202228;
}
.edit-booking .field-label {
  margin-top: 0;
}
.edit-booking label {
  display: block;
  margin: 11px 0;
  color: #b5b9c1;
  font-size: 12px;
}
.edit-booking input,
.edit-booking .field {
  box-sizing: border-box;
  width: 100%;
  margin-top: 6px;
}
.edit-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
.edit-actions button {
  flex: 1;
}
.console .picked {
  border-color: var(--lime) !important;
  background: #30381d !important;
  color: var(--lime) !important;
}
.sticky-actions {
  position: sticky;
  bottom: -24px;
  display: flex;
  gap: 10px;
  background: #1b1d22;
  padding: 16px 0;
  margin-top: 10px;
}
.sticky-actions button {
  flex: 1;
}
.error {
  display: block;
  color: #ffaaa5;
  font-size: 13px;
  margin-top: 14px;
}
.banner {
  padding: 12px;
  background: #3b2226;
  border-radius: 8px;
}
.banner text {
  display: block;
  text-decoration: underline;
  margin-top: 6px;
}
.danger {
  color: #ffaaa5 !important;
}
.refund-record {
  margin: 10px 0;
  padding: 10px 12px;
  border: 1px solid #663b42;
  border-radius: 8px;
  background: #2c2024;
}
.refund-record > text {
  display: block;
  color: #d6a3a8;
  font-size: 11px;
  margin-top: 5px;
}
.quantity {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
}
.quantity > text {
  min-width: 24px;
  text-align: center;
}
.quantity button {
  padding: 10px;
}
.quantity button:last-child {
  margin-left: auto;
}
.event {
  border-left: 1px solid #484d38;
  padding: 8px 0 8px 15px;
  font-size: 12px;
}
.event > text {
  display: block;
}
.section-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.console .text-button {
  padding: 7px 10px;
  min-height: 34px;
  color: var(--lime);
  background: transparent;
}
.audit-event {
  overflow-wrap: anywhere;
}
.audit-event view,
.audit-event > text {
  display: block;
}
.audit-event > text {
  margin-top: 4px;
}
.staff-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 11px 0;
  border-bottom: 1px solid #303239;
}
.staff-row view,
.staff-row text {
  display: block;
}
.staff-row text {
  color: #9298a2;
  font-size: 11px;
  margin-top: 4px;
}
.staff-row button {
  min-width: 64px;
  padding: 8px 10px;
}
.staff-actions {
  display: flex !important;
  flex-direction: row;
  gap: 6px;
}
.product-row {
  align-items: flex-start;
  flex-wrap: wrap;
}
.product-row .staff-actions {
  margin-left: auto;
}
.product-row .staff-actions button {
  min-width: 52px;
  padding: 8px;
}
.panel label {
  display: block;
  color: #b5b9c1;
  font-size: 12px;
  margin: 14px 0;
}
.panel label input,
.panel label .field {
  box-sizing: border-box;
  width: 100%;
  margin-top: 7px;
}
.full-button {
  width: 100%;
  margin-top: 10px;
}
.date-range {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 8px;
  margin: 14px 0;
}
.date-range .field {
  text-align: center;
  font-size: 12px;
}
.date-range > text {
  color: #777e8b;
  font-size: 12px;
}
.report-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.report-grid view {
  padding: 12px;
  border-radius: 9px;
  background: #212329;
}
.report-grid text,
.report-grid b {
  display: block;
}
.report-grid text {
  color: #9298a2;
  font-size: 11px;
}
.report-grid b {
  margin-top: 6px;
  font-size: 18px;
}
.report-days {
  margin-top: 12px;
  max-height: 230px;
  overflow-y: auto;
}
.report-days view {
  display: grid;
  grid-template-columns: 52px 1fr auto;
  gap: 8px;
  padding: 9px 0;
  border-bottom: 1px solid #303239;
  font-size: 12px;
}
.report-days text:nth-child(2) {
  color: #9298a2;
}
.filter-pickers {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 14px;
}
.filter-pickers .field {
  text-align: center;
  font-size: 12px;
}
.export-button {
  width: 100%;
  margin-top: 12px;
  border-color: #4b5232 !important;
  color: var(--lime) !important;
  background: #23271d !important;
}
.login-panel {
  padding: 60px 2px 30px;
}
.login-title {
  display: block;
  font-size: 32px;
  letter-spacing: -1px;
  font-weight: 700;
  margin: 24px 0 16px;
}
.login-card {
  margin-top: 32px;
  padding: 24px 20px;
}
.login-card input {
  margin: 20px 0 14px;
  width: 100%;
}
.role-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin: 10px 0 16px;
}
.console .role-card {
  text-align: left;
  padding: 12px;
}
.role-card b,
.role-card text {
  display: block;
}
.role-card text {
  color: #90959f;
  font-size: 11px;
  margin-top: 3px;
}
.event small {
  color: #9298a2;
  font-size: 11px;
}
.login-card button {
  width: 100%;
}
</style>

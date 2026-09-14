import { test, expect, type Page } from "@playwright/test";
const button = (page: Page, text: string) =>
  page.locator("uni-button:not([disabled])").filter({ hasText: text });
test("mobile booking, add item, checkout, release table and shared state", async ({
  page,
  browser,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/club/");
  await page
    .locator(".login-card input")
    .fill(process.env.CLUB_TEST_CODE || "local-test-only");
  await button(page, "进入工作台 →").click();
  await expect(page.getByText("让每一桌，都井然有序。")).toBeVisible();
  const customer = `自动验收${Date.now()}`;
  await button(page, "＋ 创建预订").click();
  await page
    .locator("uni-label")
    .filter({ hasText: "客户姓名" })
    .locator("input")
    .fill(customer);
  await page
    .locator("uni-label")
    .filter({ hasText: "手机号码" })
    .locator("input")
    .fill("13800000000");
  await page
    .locator("uni-label")
    .filter({ hasText: "订单备注" })
    .locator("input")
    .fill("生日客户 · 无糖饮料");
  await button(page, "下一步 · 选择台位 →").click();
  const tableID = await page.locator(".sheet .seat b").first().innerText();
  await page.locator(".sheet .seat").first().click();
  await button(page, "经典畅饮套餐").click();
  await button(page, "确认预订").click();
  await expect(button(page, "确认到店")).toBeVisible();
  await expect(
    page.getByText("生日客户 · 无糖饮料", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("4 小时", { exact: true })).toBeVisible();
  await button(page, "编辑预订资料").click();
  await page.locator(".edit-booking input").nth(3).fill("生日客户 · 靠近舞台");
  await button(page, "保存修改").click();
  await expect(page.getByText("生日客户 · 靠近舞台", { exact: true })).toBeVisible();
  await button(page, "确认到店").click();
  await button(page, "开台 · 开始服务").click();
  // Simulate the server committing a write while its response is lost in transit.
  await page.route("**/api/v1/orders/*/items", async (route) => {
    await route.fetch();
    await route.abort("failed");
  });
  await button(page, "确认加单").click();
  await expect(
    page.getByText("网络连接失败，结果可能尚未确认；请重试同一操作", {
      exact: true,
    }),
  ).toBeVisible();
  await page.unroute("**/api/v1/orders/*/items");
  await button(page, "确认加单").click();
  await expect(page.getByText("精酿啤酒 × 1", { exact: true })).toBeVisible();
  await button(page, "确认已收款 ¥2,928").click();
  await page.getByText("确定", { exact: true }).click();
  await button(page, "完成清台 · 释放台位").click();
  await expect(page.locator(".sheet .badge")).toHaveText("已完成");
  await expect(page.getByText("已收 · 微信", { exact: true })).toBeVisible();
  await button(page, "登记退款").click();
  await button(page, "确认登记").click();
  await page.getByText("确定", { exact: true }).click();
  await expect(page.getByText("累计已退款", { exact: true })).toBeVisible();
  await page.locator("#close-order").click();
  await expect(page.locator("#close-order")).toHaveCount(0);
  await page.locator(".tabbar uni-button").filter({ hasText: "台位" }).click();
  await expect(page.locator(".seat")).toHaveCount(18);
  await expect(
    page.locator(".seat.free").filter({ hasText: tableID }),
  ).toHaveCount(1);
  await expect(page.getByText("操作成功", { exact: true })).not.toBeVisible();
  await page.screenshot({
    path: "test-results/tables-mobile.png",
    fullPage: true,
  });
  await page
    .locator(".tabbar uni-button")
    .filter({ hasText: "工作台" })
    .click();
  await page.screenshot({
    path: "test-results/dashboard-mobile.png",
    fullPage: true,
  });
  const second = await browser.newContext({
    viewport: { width: 375, height: 812 },
  });
  const p2 = await second.newPage();
  await p2.goto(
    (process.env.CLUB_TEST_URL || "http://127.0.0.1:18080") + "/club/",
  );
  await p2
    .locator(".login-card input")
    .fill(process.env.CLUB_TEST_CODE || "local-test-only");
  await button(p2, "进入工作台 →").click();
  await p2.locator(".tabbar uni-button").filter({ hasText: "订单" }).click();
  await p2.locator(".search input").fill(customer);
  await expect(p2.locator(".order-card")).toHaveCount(1);
  await expect(p2.locator(".order-card .badge")).toHaveText("已完成");
  await second.close();
  for (const width of [320, 375, 480]) {
    await page.setViewportSize({ width, height: 812 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBeTruthy();
  }
  expect(errors).toEqual([]);
});

test("employee role changes visible actions", async ({ page }) => {
  await page.goto("/club/");
  await page.locator(".role-card").filter({ hasText: "服务员阿杰" }).click();
  await page
    .locator(".login-card input")
    .fill(process.env.CLUB_TEST_CODE || "local-test-only");
  await button(page, "进入工作台 →").click();
  await expect(page.getByText("让每一桌，都井然有序。")).toBeVisible();
  await expect(button(page, "＋ 创建预订")).toHaveCount(0);
  await page.locator(".tabbar uni-button").filter({ hasText: "我的" }).click();
  await expect(page.getByText("服务员阿杰", { exact: true })).toBeVisible();
  await expect(
    page.getByText("确认到店、开台、加单、结账、清台", { exact: true }),
  ).toBeVisible();
});

test("owner can see database-backed employee management", async ({ page }) => {
  await page.goto("/club/");
  await page.locator(".role-card").filter({ hasText: "店长" }).click();
  await page
    .locator(".login-card input")
    .fill(process.env.CLUB_TEST_CODE || "local-test-only");
  await button(page, "进入工作台 →").click();
  await page.locator(".tabbar uni-button").filter({ hasText: "我的" }).click();
  await expect(page.getByText("营业报表", { exact: true })).toBeVisible();
  await expect(page.getByText("全店数据", { exact: true })).toBeVisible();
  const reportDownload = page.waitForEvent("download");
  await button(page, "导出报表 CSV").click();
  expect((await reportDownload).suggestedFilename()).toContain("营业报表");
  await expect(page.locator(".filter-pickers .field").nth(0)).toContainText("全部员工");
  await expect(page.locator(".filter-pickers .field").nth(1)).toContainText("全部操作");
  const auditDownload = page.waitForEvent("download");
  await button(page, "导出审计 CSV").click();
  expect((await auditDownload).suggestedFilename()).toContain("操作审计");
  await expect(page.getByText("员工账号", { exact: true })).toBeVisible();
  await expect(page.locator(".staff-row b").filter({ hasText: "销售小林" })).toBeVisible();
  await expect(button(page, "创建员工账号")).toBeVisible();
  await expect(page.getByText("商品与套餐", { exact: true })).toBeVisible();
  await expect(page.locator(".product-row b").filter({ hasText: "经典畅饮套餐" })).toBeVisible();
  await expect(button(page, "创建并上架")).toBeVisible();
});

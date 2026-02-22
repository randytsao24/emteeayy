import { test, expect, Page } from "@playwright/test";

// ---------------------------------------------------------------------------
// Shared mock API responses
// ---------------------------------------------------------------------------

const MOCK_SUBWAY_RESPONSE = {
  success: true,
  zip_code: "10001",
  radius_meters: 800,
  count: 1,
  stations: [
    {
      stop_id: "127",
      stop_name: "Times Sq-42 St",
      stop_lat: 40.75529,
      stop_lon: -73.987495,
      distance_meters: 962,
      distance_miles: 0.6,
      northbound: [
        { route: "A", stop_id: "127N", direction: "northbound", minutes_away: 3 },
        { route: "C", stop_id: "127N", direction: "northbound", minutes_away: 7 },
      ],
      southbound: [
        { route: "A", stop_id: "127S", direction: "southbound", minutes_away: 5 },
      ],
    },
  ],
};

const MOCK_BUS_RESPONSE = {
  success: true,
  zip_code: "10001",
  radius_meters: 400,
  count: 2,
  arrivals: [
    {
      route: "M34",
      destination: "34 St Ferry",
      stop_id: "MTA_305423",
      stop_name: "5 AV/W 34 ST",
      minutes_away: 3,
    },
    {
      route: "M34A",
      destination: "34 St-Hudson Yards",
      stop_id: "MTA_305423",
      stop_name: "5 AV/W 34 ST",
      minutes_away: 8,
    },
  ],
};

const MOCK_ERROR_RESPONSE = {
  success: false,
  error: "Zip code not found",
  message: "Zip code 00000 is not in our NYC database",
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function mockSubwayAPI(page: Page, response = MOCK_SUBWAY_RESPONSE) {
  await page.route(/\/transit\/subway\/near\/.*/, async (route) => {
    await route.fulfill({ json: response });
  });
}

async function mockBusAPI(page: Page, response = MOCK_BUS_RESPONSE) {
  await page.route(/\/transit\/bus\/near\/.*/, async (route) => {
    await route.fulfill({ json: response });
  });
}

async function enterZipAndSearch(page: Page, zip: string) {
  await page.fill('input[type="text"]', zip);
  await page.click('button[type="submit"]');
}

// ---------------------------------------------------------------------------
// Page load
// ---------------------------------------------------------------------------

test("page loads with correct title and header", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/emteeayy/i);
  await expect(page.locator("h1")).toContainText("emteeayy");
  await expect(page.locator(".tagline")).toBeVisible();
});

test("subway tab is active by default", async ({ page }) => {
  await page.goto("/");
  const subwayTab = page.locator(".tab", { hasText: "Subway" });
  await expect(subwayTab).toHaveClass(/active/);
  const busTab = page.locator(".tab", { hasText: "Bus" });
  await expect(busTab).not.toHaveClass(/active/);
});

test("search form is visible on load", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator('input[type="text"]')).toBeVisible();
  await expect(page.locator('button[type="submit"]')).toBeVisible();
  await expect(page.locator(".btn-secondary")).toBeVisible();
});

test("radius dropdown has expected options", async ({ page }) => {
  await page.goto("/");
  const select = page.locator(".radius-select");
  await expect(select).toBeVisible();
  await expect(select.locator("option")).toHaveCount(4);
});

// ---------------------------------------------------------------------------
// Zip code input validation
// ---------------------------------------------------------------------------

test("zip input enforces max 5 characters", async ({ page }) => {
  await page.goto("/");
  const input = page.locator('input[type="text"]');
  await input.fill("123456789");
  const value = await input.inputValue();
  expect(value.length).toBeLessThanOrEqual(5);
});

test("zip input has correct pattern attribute", async ({ page }) => {
  await page.goto("/");
  const input = page.locator('input[type="text"]');
  const pattern = await input.getAttribute("pattern");
  expect(pattern).toBe("[0-9]{5}");
});

test("submitting empty zip does not trigger API call", async ({ page }) => {
  let apiCalled = false;
  await page.route(/\/transit\/subway\/near\/.*/, async (route) => {
    apiCalled = true;
    await route.fulfill({ json: MOCK_SUBWAY_RESPONSE });
  });

  await page.goto("/");
  await page.click('button[type="submit"]');
  await page.waitForTimeout(500);
  expect(apiCalled).toBe(false);
});

// ---------------------------------------------------------------------------
// Subway search results
// ---------------------------------------------------------------------------

test("valid zip search shows subway stations", async ({ page }) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".station")).toBeVisible({ timeout: 5000 });
  await expect(page.locator(".station-name").first()).toContainText(
    "Times Sq-42 St"
  );
});

test("subway result shows direction labels", async ({ page }) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".direction-label").first()).toBeVisible({
    timeout: 5000,
  });
});

test("subway result shows line badges", async ({ page }) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".line-badge").first()).toBeVisible({
    timeout: 5000,
  });
});

test("shows last updated timestamp after search", async ({ page }) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".last-updated")).toBeVisible({ timeout: 5000 });
});

// ---------------------------------------------------------------------------
// Error states
// ---------------------------------------------------------------------------

test("API error response shows error message", async ({ page }) => {
  await page.route(/\/transit\/subway\/near\/.*/, async (route) => {
    await route.fulfill({ status: 404, json: MOCK_ERROR_RESPONSE });
  });

  await page.goto("/");
  await enterZipAndSearch(page, "00000");

  await expect(page.locator(".error")).toBeVisible({ timeout: 5000 });
});

test("network failure shows error message", async ({ page }) => {
  await page.route(/\/transit\/subway\/near\/.*/, async (route) => {
    await route.abort("failed");
  });

  await page.goto("/");
  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".error")).toBeVisible({ timeout: 5000 });
});

// ---------------------------------------------------------------------------
// Tab switching
// ---------------------------------------------------------------------------

test("clicking Bus tab makes it active", async ({ page }) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await page.click('.tab:has-text("Bus")');

  const busTab = page.locator(".tab", { hasText: "Bus" });
  await expect(busTab).toHaveClass(/active/);
  const subwayTab = page.locator(".tab", { hasText: "Subway" });
  await expect(subwayTab).not.toHaveClass(/active/);
});

test("switching to Bus tab and searching shows bus results", async ({
  page,
}) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await page.click('.tab:has-text("Bus")');
  await enterZipAndSearch(page, "10001");

  await expect(page.locator(".bus-stop")).toBeVisible({ timeout: 5000 });
});

test("switching back to Subway tab after Bus shows subway content area", async ({
  page,
}) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);
  await page.goto("/");

  await page.click('.tab:has-text("Bus")');
  await page.click('.tab:has-text("Subway")');

  const subwayTab = page.locator(".tab", { hasText: "Subway" });
  await expect(subwayTab).toHaveClass(/active/);
});

// ---------------------------------------------------------------------------
// Location button
// ---------------------------------------------------------------------------

test("'Use my location' button is visible and enabled by default", async ({
  page,
}) => {
  await page.goto("/");
  const btn = page.locator(".btn-secondary");
  await expect(btn).toBeVisible();
  await expect(btn).not.toBeDisabled();
  await expect(btn).toContainText("Use my location");
});

test("clicking location button triggers geolocation request", async ({
  page,
}) => {
  await mockSubwayAPI(page);
  await mockBusAPI(page);

  await page.context().grantPermissions(["geolocation"]);
  await page.context().setGeolocation({ latitude: 40.7484, longitude: -73.9967 });

  // Also mock coordinate-based endpoint
  await page.route(/\/transit\/subway\/near\?.*/, async (route) => {
    await route.fulfill({ json: { ...MOCK_SUBWAY_RESPONSE, zip_code: undefined } });
  });

  await page.goto("/");
  await page.click(".btn-secondary");

  // Proves geolocation triggered (Alpine sets locationText after coords are obtained)
  await expect(page.locator(".location-info")).toContainText("Near", { timeout: 5000 });
  // Proves the search completed and returned results
  await expect(page.locator(".station").first()).toBeVisible({ timeout: 5000 });
});

// ---------------------------------------------------------------------------
// Privacy note
// ---------------------------------------------------------------------------

test("privacy note is present", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".privacy-note")).toBeVisible();
  await expect(page.locator(".privacy-note")).toContainText(
    "does NOT collect"
  );
});

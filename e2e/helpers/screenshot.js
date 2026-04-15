'use strict';

/**
 * Take a screenshot and attach it to the Playwright test report.
 *
 * @param {import('@playwright/test').Page}     page
 * @param {import('@playwright/test').TestInfo}  testInfo
 * @param {string}                               path - file path for the PNG
 */
async function takeAndAttachScreenshot(page, testInfo, path) {
    await page.screenshot({ path: path, animations: 'disabled' });
    testInfo.attachments.push({
        name: 'Screenshot',
        path: path,
        contentType: 'image/png',
    });
}

/**
 * Take a screenshot of a specific locator and attach it to the report.
 *
 * @param {import('@playwright/test').Locator}   locator
 * @param {import('@playwright/test').TestInfo}  testInfo
 * @param {string}                               path - file path for the PNG
 */
async function screenshotLocator(locator, testInfo, path) {
    await locator.screenshot({ path: path, animations: 'disabled' });
    testInfo.attachments.push({
        name: 'Screenshot',
        path: path,
        contentType: 'image/png',
    });
}

module.exports = { takeAndAttachScreenshot, screenshotLocator };

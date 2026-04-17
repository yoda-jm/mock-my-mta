'use strict';

const { SearchSection }          = require('../sections/SearchSection');
const { EmailListSection }       = require('../sections/EmailListSection');
const { EmailViewSection }       = require('../sections/EmailViewSection');
const { ReleaseModalSection }    = require('../sections/ReleaseModalSection');
const { MailboxSection }         = require('../sections/MailboxSection');
const { SyntaxHelpModalSection } = require('../sections/SyntaxHelpModalSection');

/**
 * Top-level page object for the mock-my-mta single-page application.
 *
 * The app has two panes:
 *   - email list  (search + table + pagination)
 *   - email view  (detail, body tabs, attachments)
 *
 * Both are always in the DOM; visibility toggles between them.
 * All sections are available upfront — use the locators' .isVisible() /
 * toBeVisible() / toBeHidden() in tests to assert which pane is active.
 *
 * Usage:
 *   const inbox = new InboxPage(page);
 *   await inbox.goto();
 *   await inbox.search.search('hello');
 *   await inbox.emailList.firstRow().open();
 *   await expect(inbox.emailView.locator).toBeVisible();
 */
class InboxPage {
  constructor(page) {
    this.page         = page;
    this.search       = new SearchSection(page);
    this.emailList    = new EmailListSection(page);
    this.emailView    = new EmailViewSection(page);
    this.releaseModal = new ReleaseModalSection(page);
    this.mailbox      = new MailboxSection(page);
    this.syntaxHelp   = new SyntaxHelpModalSection(page);
  }

  async goto() {
    await this.page.goto('/');
    await this.page.waitForSelector('[data-testid="email-list-body"]');
  }

  /**
   * Navigate directly to an email by ID using the deep link URL.
   * @param {string} emailId
   */
  async gotoEmail(emailId) {
    await this.page.goto(`/#/email/${encodeURIComponent(emailId)}`);
    await this.page.waitForSelector('.email-view', { state: 'visible', timeout: 10000 });
  }

  /**
   * Navigate directly to a search query using the deep link URL.
   * @param {string} query
   */
  async gotoSearch(query) {
    await this.page.goto(`/#/search/${encodeURIComponent(query)}`);
    await this.page.waitForSelector('[data-testid="email-list-body"]');
  }

  /**
   * Call the wait-for-email API endpoint.
   * Long-polls until a matching email arrives or timeout.
   *
   * @param {string} query   Search query (same syntax as the search box)
   * @param {string} timeout Duration string (e.g. '5s', '30s')
   * @returns {Promise<{email: object, total_matches: number, url: string}>}
   */
  async waitForEmail(query, timeout = '10s') {
    const baseUrl = this.page.url().replace(/#.*$/, '').replace(/\/$/, '');
    const url = `${baseUrl}/api/emails/wait?query=${encodeURIComponent(query)}&timeout=${timeout}`;
    const response = await this.page.request.get(url);
    if (!response.ok()) {
      throw new Error(`waitForEmail failed (${response.status()}): ${await response.text()}`);
    }
    return response.json();
  }
}

module.exports = { InboxPage };

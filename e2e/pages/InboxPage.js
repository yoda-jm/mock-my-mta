'use strict';

const { SearchSection }      = require('../sections/SearchSection');
const { EmailListSection }   = require('../sections/EmailListSection');
const { EmailViewSection }   = require('../sections/EmailViewSection');
const { ReleaseModalSection } = require('../sections/ReleaseModalSection');

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
  }

  async goto() {
    await this.page.goto('/');
    await this.page.waitForSelector('[data-testid="email-list-body"]');
  }
}

module.exports = { InboxPage };

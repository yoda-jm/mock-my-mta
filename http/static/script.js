$(function () {
    let currentEmailId = null;
    let selectedEmailIds = new Set();
    let lastKnownEmailCount = null;
    let pollingInterval = null;

    // ── Dark mode ──────────────────────────────────────────────────────
    function initTheme() {
        const saved = localStorage.getItem('theme');
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        const theme = saved || (prefersDark ? 'dark' : 'light');
        applyTheme(theme);
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
        const icon = $('#theme-toggle i');
        const label = $('#theme-toggle .theme-toggle-label');
        if (theme === 'dark') {
            icon.removeClass('bi-moon-fill').addClass('bi-sun-fill');
            label.text('Light mode');
        } else {
            icon.removeClass('bi-sun-fill').addClass('bi-moon-fill');
            label.text('Dark mode');
        }
    }

    $('#theme-toggle').click(function () {
        const current = document.documentElement.getAttribute('data-theme') || 'light';
        applyTheme(current === 'dark' ? 'light' : 'dark');
    });

    initTheme();

    const searchInput = $('.search-box input[type="text"]');
    const suggestionDisplay = $('#suggestion-display');


    searchInput.on('keyup', function (e) {
        // Ignore arrow keys, shift, ctrl, alt, meta, escape, enter for suggestion logic
        // Arrow keys, shift, ctrl, alt, meta, escape, enter are handled by keydown or ignored for suggestion purposes
        if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight',
             'Shift', 'Control', 'Alt', 'Meta'].includes(e.key)) {
            // Escape and Enter are handled in keydown
            return;
        }

        const fullText = $(this).val();
        const cursorPos = this.selectionStart; // `this` is the input element
        const textBeforeCursor = fullText.substring(0, cursorPos);

        const lastSpace = textBeforeCursor.lastIndexOf(' ');
        const tokenStart = (lastSpace === -1) ? 0 : lastSpace + 1;
        const currentToken = textBeforeCursor.substring(tokenStart);

        if (currentToken.trim() === '') {
            suggestionDisplay.text(''); // MODIFIED
            return;
        }

        $.ajax({
            url: `/api/filters/suggestions?term=${encodeURIComponent(currentToken)}`,
            type: 'GET',
            success: function (data) {
                if (data && data.length > 0) {
                    const firstSuggestionCandidate = data[0]; // This is the suggestion for currentToken
                    const textBeforeToken = fullText.substring(0, tokenStart);

                    // Only show suggestion if it's longer than current token and starts with it (case insensitive)
                    if (firstSuggestionCandidate.toLowerCase().startsWith(currentToken.toLowerCase()) && currentToken.length < firstSuggestionCandidate.length) {
                        const fullSuggestedText = textBeforeToken + firstSuggestionCandidate;
                        suggestionDisplay.text(fullSuggestedText);
                    } else {
                        suggestionDisplay.text(''); // No valid suggestion or current token already matches suggestion
                    }
                } else {
                    suggestionDisplay.text(''); // No suggestions from API
                }
            },
            error: function (jqXHR, textStatus, errorThrown) {
                console.error('Error fetching suggestions:', textStatus, errorThrown);
                suggestionDisplay.text(''); // MODIFIED
            }
        });
    });

    searchInput.on('keydown', function(e) {
        const currentSuggestionText = suggestionDisplay.text();
        // Check if suggestion is available and visible (not empty) and different from current input
        if (currentSuggestionText && currentSuggestionText !== $(this).val()) {
            if (e.key === 'Tab') {
                e.preventDefault(); // Prevent default tab behavior
                $(this).val(currentSuggestionText); // Set input to the full suggestion
                suggestionDisplay.text(''); // Clear the suggestion display

                // Move cursor to the end of the input
                const inputElem = $(this)[0];
                if (inputElem.setSelectionRange) {
                    inputElem.setSelectionRange(currentSuggestionText.length, currentSuggestionText.length);
                } else if (inputElem.createTextRange) { // IE
                    const range = inputElem.createTextRange();
                    range.collapse(true);
                    range.moveEnd('character', currentSuggestionText.length);
                    range.moveStart('character', currentSuggestionText.length);
                    range.select();
                }
            } else if (e.key === 'Escape') {
                suggestionDisplay.text('');
                 // Prevent event from bubbling up to other escape listeners if any (e.g. modal)
                e.stopPropagation();
            }
        } else if (e.key === 'Escape') { // If no suggestion or suggestion is same as input, Escape should still clear display
             suggestionDisplay.text('');
             e.stopPropagation();
        }

        // If Enter is pressed, clear suggestion and let the existing keypress handler for Enter submit the search
        if (e.key === 'Enter' && suggestionDisplay.text() !== '') {
            suggestionDisplay.text('');
        }
    });

    searchInput.on('blur', function () {
        setTimeout(function() { suggestionDisplay.text(''); }, 150);
    });

    // Event listener for the syntax help icon
    $('#show-syntax-help').on('click', function () {
        $.ajax({
            url: '/api/filters/suggestions', // No 'term' parameter
            type: 'GET',
            success: function (data) {
                const syntaxHelpTableBody = $('#syntaxHelpTableBody');
                syntaxHelpTableBody.empty(); // Clear previous content

                if (data && data.length > 0) {
                    data.forEach(function (entry) {
                        const row = $('<tr>');
                        row.append($('<td>').text(entry.command));
                        row.append($('<td>').text(entry.suggestion));
                        row.append($('<td>').text(entry.description));
                        syntaxHelpTableBody.append(row);
                    });
                } else {
                    // Optionally, display a message if no syntax help is available
                    syntaxHelpTableBody.append('<tr><td colspan="3" class="text-center">No syntax help available.</td></tr>');
                }

                // Show the modal
                const helpModal = new bootstrap.Modal($('#syntaxHelpModal')[0]);
                helpModal.show();
            },
            error: function (jqXHR, textStatus, errorThrown) {
                console.error('Error fetching syntax help:', textStatus, errorThrown);
                showPopup('Could not load syntax help: ' + errorThrown, 'error');
            }
        });
    });

    // initialize tooltips
    $('[title]').tooltip();
    // initialize the search
    setSearchQuery('');
    resetCurrentPage()
    refreshEmailList();
    startPolling();

    $('.bi-arrow-left').click(function () {
        displayEmailList();
    });

    $('[data-testid="email-view-release-button"]').click(function () {
        if (currentEmailId) {
            displayReleaseModal(currentEmailId);
        }
    });

    $('[data-testid="email-view-download-button"]').click(function () {
        if (currentEmailId) {
            window.location.href = '/api/emails/' + currentEmailId + '/download';
        }
    });

    $('[data-testid="email-view-delete-button"]').click(function () {
        if (currentEmailId) {
            deleteEmail(currentEmailId);
            currentEmailId = null;
        }
    });

    // Function to process images based on the toggle state
    function processEmailImages(shadowRoot) {
        if (!shadowRoot) return;

        const displayExternal = $('#displayExternalImagesToggle').is(':checked');
        const images = shadowRoot.querySelectorAll('img');

        images.forEach(img => {
            const originalSrc = img.getAttribute('src');
            // Check if the src exists and starts with http:// or https://
            if (originalSrc && (originalSrc.startsWith('http://') || originalSrc.startsWith('https://'))) {
                if (displayExternal) {
                    img.style.display = ''; // Show the image
                } else {
                    img.style.display = 'none'; // Hide the image
                }
            }
        });
    }

    // Event listener for the toggle switch
    $('#displayExternalImagesToggle').on('change', function() {
        const host = document.querySelectorAll('.email-content')[0];
        if (host && host.shadowRoot) {
            processEmailImages(host.shadowRoot);
        }
    });

    function displayEmailList() {
        $('.email-view').hide();
        $('.email-list').show();
    }

    function displayEmailView() {
        $('.email-list').hide();
        $('.email-view').show();
    }

    function displayReleaseModal(emailId) {
        $.ajax({
            url: '/api/emails/' + emailId + '/relay',
            type: 'GET',
            success: function (data) {
                // log data
                console.log(data);
                const modal = $('#releaseEmailModal');
                // fill the modal with the data
                $('#emailId').val(emailId);
                $('#originalSender').val(data.sender.address);
                $('#originalReceivers').val(data.recipients.map(function (recipient) { return recipient.address; }).join(', '));
                // clear the relay configs select options
                $('#relayConfig').empty();
                for (var i = 0; i < data.relay_names.length; i++) {
                    var configName = data.relay_names[i];
                    $('#relayConfig').append($('<option>').val(configName).text(configName));
                }
                // display modal
                modal.modal('show');
            },
        });
    }

    $('#releaseEmailButton').on('click', function() {
        var emailId = $('#emailId').val();
        var relayName = $('#relayConfig').val();
        var sender = '';
        if ($('#senderOverride').is(':checked')) {
            sender = $('#overrideSender').val();
        } else {
            sender = $('#originalSender').val();
        }
        var recipients = [];
        if ($('#receiversOverride').is(':checked')) {
            recipients = $('#overrideReceivers').val().split(',').map(function (recipient) { return recipient.trim(); });
        } else {
            recipients = $('#originalReceivers').val().split(',').map(function (recipient) { return recipient.trim(); });
        }
        var formData = {
            relay_name: relayName,
            sender: sender,
            recipients: recipients
        };
        $.ajax({
            url: '/api/emails/' + emailId + '/relay',
            type: 'POST',
            data: JSON.stringify(formData),
            success: function (data) {
                // log data
                console.log(data);
                // close the modal
                $('#releaseEmailModal').modal('hide');
            },
            error: function (jqXHR, textStatus, errorThrown ) {
                showPopup(jqXHR.responseText, 'error');
            }
        });
    });

    // Enable/Disable override sender input based on radio selection
    $('input[name="senderOption"]').on('change', function() {
        if ($('#senderOverride').is(':checked')) {
            $('#overrideSender').prop('disabled', false);
        } else {
            $('#overrideSender').prop('disabled', true).val('');
        }
    });

    // Enable/Disable override receivers input based on radio selection
    $('input[name="receiversOption"]').on('change', function() {
        if ($('#receiversOverride').is(':checked')) {
            $('#overrideReceivers').prop('disabled', false);
        } else {
            $('#overrideReceivers').prop('disabled', true).val('');
        }
    });

    $('.bi-x-lg').click(function () {
        setSearchQuery('');
        resetCurrentPage()
        refreshEmailList();
    });

    $('.search-box i').click(function () {
        var query = $('.search-box input').val();
        updateSearchBoxAndRefreshEmailList(query);
    });

    $('.search-box input').keypress(function (e) {
        if (e.which == 13) {
            var query = $('.search-box input').val();
            updateSearchBoxAndRefreshEmailList(query);
        }
    });

    $('[data-bs-toggle="collapse"]').click(function () {
        if ($(this).find('.icon').hasClass('bi-chevron-right')) {
            refreshMailboxes();
            // replace with a refresh icon
            $(this).find('.icon').toggleClass('bi-chevron-right').toggleClass('bi-chevron-down');
            displayMailboxes();
        } else if ($(this).find('.icon').hasClass('bi-chevron-down')) {
            // replace with a collapse icon
            $(this).find('.icon').toggleClass('bi-chevron-down').toggleClass('bi-chevron-right');
            $('#mailboxList').empty();
        }
    });

    $('#refresh').click(function () {
        // Refresh the email list
        console.log('Refreshing emails');
        refreshEmailList();
    });

    $('#allEmails').click(function () {
        // Update the search box and refresh the email list
        console.log('Displaying all emails');
        // set current page to 1
        resetCurrentPage()
        updateSearchBoxAndRefreshEmailList('');
    });

    $('#prev-page').click(function () {
        // Decrement page number and refresh email list
        var page = parseInt($('#page-start').text());
        console.log('Going to previous page');
        $('#page-start').text(page - 1);
        refreshEmailList();
    });

    $('#next-page').click(function () {
        // Increment page number and refresh email list
        var page = parseInt($('#page-start').text());
        console.log('Going to next page');
        $('#page-start').text(page + 1);
        refreshEmailList();
    });

    $('#deleteAll').click(function () {
        if (!confirm('Delete ALL emails? This cannot be undone.')) return;
        console.log('Deleting all emails');
        $.ajax({
            url: '/api/emails/',
            type: 'DELETE',
            success: function (data) {
                // refresh the email list
                console.log('All emails deleted');
                refreshMailboxes();
                refreshEmailList();
                displayEmailList();
            },
            error: function (error) {
                console.log('Error deleting emails');
                console.log(error);
            }
        });
    });

    // ── Real-time WebSocket notifications ─────────────────────────────
    function connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = protocol + '//' + window.location.host + '/api/ws';
        const ws = new WebSocket(wsUrl);

        ws.onmessage = function (event) {
            const data = JSON.parse(event.data);
            switch (data.type) {
                case 'new_email':
                    showPopup('New email received', 'info');
                    refreshEmailList();
                    break;
                case 'delete_email':
                    refreshEmailList();
                    break;
                case 'delete_all':
                    refreshEmailList();
                    break;
            }
        };

        ws.onclose = function () {
            // Reconnect after 3 seconds
            setTimeout(connectWebSocket, 3000);
        };

        ws.onerror = function () {
            ws.close();
        };
    }

    function startPolling() {
        // WebSocket is primary; polling is fallback
        try {
            connectWebSocket();
        } catch (e) {
            // Fallback to polling if WebSocket fails
            pollingInterval = setInterval(function () {
                $.ajax({
                    url: '/api/emails/',
                    type: 'GET',
                    data: { page: 1 },
                    success: function (data) {
                        const currentCount = data.pagination.total_matches;
                        if (lastKnownEmailCount !== null && currentCount > lastKnownEmailCount) {
                            const newCount = currentCount - lastKnownEmailCount;
                            showPopup(newCount + ' new email' + (newCount > 1 ? 's' : '') + ' received', 'info');
                            refreshEmailList();
                        }
                        lastKnownEmailCount = currentCount;
                    }
                });
            }, 5000);
        }
    }

    // ── Bulk operations ──────────────────────────────────────────────────
    function updateBulkToolbar() {
        if (selectedEmailIds.size > 0) {
            $('#bulk-toolbar').css('display', 'flex');
            $('#bulk-count').text(selectedEmailIds.size + ' selected');
        } else {
            $('#bulk-toolbar').css('display', 'none');
        }
    }

    function clearSelection() {
        selectedEmailIds.clear();
        $('.email-checkbox').prop('checked', false);
        $('#select-all-checkbox').prop('checked', false);
        updateBulkToolbar();
    }

    $(document).on('change', '.email-checkbox', function (e) {
        e.stopPropagation();
        const emailId = $(this).data('email-id');
        if ($(this).is(':checked')) {
            selectedEmailIds.add(emailId);
        } else {
            selectedEmailIds.delete(emailId);
        }
        updateBulkToolbar();
    });

    $(document).on('change', '#select-all-checkbox', function () {
        const checked = $(this).is(':checked');
        selectedEmailIds.clear();
        if (checked) {
            $('.email-checkbox').each(function () {
                $(this).prop('checked', true);
                selectedEmailIds.add($(this).data('email-id'));
            });
        } else {
            $('.email-checkbox').prop('checked', false);
        }
        updateBulkToolbar();
    });

    $('#bulk-delete').click(function () {
        if (selectedEmailIds.size === 0) return;
        if (!confirm('Delete ' + selectedEmailIds.size + ' email(s)?')) return;
        $.ajax({
            url: '/api/emails/bulk-delete',
            type: 'POST',
            contentType: 'application/json',
            data: JSON.stringify({ ids: Array.from(selectedEmailIds) }),
            success: function (data) {
                showPopup(data.succeeded.length + ' email(s) deleted', 'success');
                if (data.failed && data.failed.length > 0) {
                    showPopup(data.failed.length + ' email(s) failed to delete', 'warning');
                }
                clearSelection();
                refreshEmailList();
            },
            error: function () {
                showPopup('Bulk delete failed', 'error');
            }
        });
    });

    $('#bulk-release').click(function () {
        if (selectedEmailIds.size === 0) return;
        // Open release modal but wire it up for bulk relay
        $.ajax({
            url: '/api/emails/' + Array.from(selectedEmailIds)[0] + '/relay',
            type: 'GET',
            success: function (data) {
                const modal = $('#releaseEmailModal');
                $('#emailId').val(selectedEmailIds.size + ' emails selected');
                $('#originalSender').val(data.sender.address);
                $('#originalReceivers').val(data.recipients.map(r => r.address).join(', '));
                $('#relayConfig').empty();
                for (const configName of data.relay_names) {
                    $('#relayConfig').append($('<option>').val(configName).text(configName));
                }
                // Override the release button for bulk
                $('#releaseEmailButton').off('click').on('click', function () {
                    const relayName = $('#relayConfig').val();
                    let sender = $('#senderOverride').is(':checked') ? $('#overrideSender').val() : $('#originalSender').val();
                    let recipients = $('#receiversOverride').is(':checked')
                        ? $('#overrideReceivers').val().split(',').map(r => r.trim())
                        : $('#originalReceivers').val().split(',').map(r => r.trim());
                    $.ajax({
                        url: '/api/emails/bulk-relay',
                        type: 'POST',
                        contentType: 'application/json',
                        data: JSON.stringify({
                            ids: Array.from(selectedEmailIds),
                            relay_name: relayName,
                            sender: sender,
                            recipients: recipients
                        }),
                        success: function (result) {
                            showPopup(result.succeeded.length + ' email(s) relayed', 'success');
                            if (result.failed && result.failed.length > 0) {
                                showPopup(result.failed.length + ' email(s) failed', 'warning');
                            }
                            modal.modal('hide');
                            clearSelection();
                        },
                        error: function (jqXHR) {
                            showPopup(jqXHR.responseText, 'error');
                        }
                    });
                });
                modal.modal('show');
            }
        });
    });

    function refreshMailboxes() {
        // Load mailboxes
        console.log('Refreshing mailboxes');
        // ajax call to retreive mailboxes
        $.ajax({
            url: '/api/mailboxes',
            type: 'GET',
            success: function (data) {
                // clear mailboxList
                $('#mailboxList').empty();
                for (var i = 0; i < data.length; i++) {
                    // Add mailbox to mailboxList, allow text to overflow
                    var mailbox = data[i];
                    $('#mailboxList').append(generateMailboxListItem(mailbox));
                }
            },
            error: function (error) {
                console.log('Error retrieving mailboxes');
                console.log(error);
            }
        });
    }

    function displayMailboxes() {
        // Display mailboxes
        console.log('Displaying mailboxes');
        $('#mailboxList').show();
    }

    function updateSearchBoxAndRefreshEmailList(query) {
        // Update the search box and submit the form
        console.log('Updating search box and submitting form');
        suggestionDisplay.text(''); // MODIFIED
        setSearchQuery(query);
        resetCurrentPage()
        refreshEmailList()
        displayEmailList();
    }

    function refreshEmailList() {
        clearSelection();
        const query = $('.search-box input').val();
        const page = $('#page-start').text();
        $.ajax({
            url: '/api/emails/',
            type: 'GET',
            data: {
                query: query,
                page: page
            },
            success: function (data) {
                // update the email list and pagination
                console.log('Updating email list');
                const emails = data.emails;
                const emailList = $('.email-list .email-table tbody');
                emailList.empty();
                if (emails == null || emails.length == 0) {
                    emailList.append(generateEmptyEmailListItem());
                    updatePagination(data.pagination);
                    return;
                }
                for (var i = 0; i < emails.length; i++) {
                    var email = emails[i];
                    emailList.append(generateEmailListItem(email));
                }
                updatePagination(data.pagination);
            },
            error: function (error) {
                console.log('Error retrieving emails');
                console.log(error);
            }
        });
    }

    function updateEmailContentHeader(email) {
        // Update the email content header
        console.log('Updating email content header');
        $('.email-header').empty();
        console.log(email);
        $('.email-header').append($('<p>').append($('<strong>').text('ID: ')).append(email.id));
        $('.email-header').append($('<p>').append($('<strong>').text('Date: ')).append(formatDateTime(email.date)));
        $('.email-header').append($('<p>').append($('<strong>').text('From: ')).append(formatEmailAddress(email.from)));
        $('.email-header').append($('<p>').append($('<strong>').text('To: ')).append(formatEmailAddresses(email.tos)));
        $('.email-header').append($('<p>').append($('<strong>').text('CC: ')).append(formatEmailAddresses(email.ccs)));
        $('.email-header').append($('<p>').append($('<strong>').text('Subject: ')).append(email.subject));
    }

    function updateEmailAttachments(emailId) {
        // Update the email attachments
        console.log('Updating email attachments');
        $.ajax({
            url: '/api/emails/' + emailId + '/attachments/',
            type: 'GET',
            success: function (data) {
                // update the email attachments
                console.log('Updating email attachments');
                const attachments = data;
                const attachmentList = $('.email-attachments');
                attachmentList.empty();
                if (attachments == null || attachments.length == 0) {
                    return;
                }
                const attachmentsHtml = $('<span>');
                for (var i = 0; i < attachments.length; i++) {
                    var attachment = attachments[i];
                    // I have content_type, size, filename, id, coma separated links in spans on the same line
                    var attachmentLink = $('<a>').attr('href', '/api/emails/' + emailId + '/attachments/' + attachment.id + '/content')
                                                 .text(attachment.filename)
                                                 .attr('data-testid', `attachment-link-${attachment.id}`);
                    var attachmentHtml = $('<span>').append(attachmentLink);
                    attachmentHtml.append(' (' + attachment.size +' bytes)');
                    if (i < attachments.length - 1) {
                        attachmentHtml.append(', ');
                    }
                    attachmentsHtml.append(attachmentHtml);
                }
                attachmentList.append($('<p>').append($('<strong>').text('Attachments: ')).append(attachmentsHtml));
            }
        });
    }

    function openShadowRootNotExisting() {
        // Open the shadow root if it does not exist
        const host = document.querySelectorAll('.email-content')[0];
        let shadowRoot = host.shadowRoot;
        if (!shadowRoot) {
            // If no shadow root, attach it
            shadowRoot = host.attachShadow({ mode: 'open' });
        }
        return shadowRoot;
    }

    function renderEmailBody(selectedBodyVersion, data) {
        console.log('Rendering email body ' + selectedBodyVersion + ' version');
        hideExtraPanels();
        $('.email-content').empty();
        var shadowRoot = openShadowRootNotExisting();
        // FIXME: why is the CSP not working?
        // shadowRoot.innerHTML = '<meta http-equiv="Content-Security-Policy" content="default-src \'self\'">';
        shadowRoot.innerHTML = ""; // Clear previous content before adding new

        if (selectedBodyVersion == 'raw') {
            const pre = document.createElement('pre');
            pre.textContent = data;
            shadowRoot.appendChild(pre);
        } else if (selectedBodyVersion == 'html' || selectedBodyVersion == 'watch-html') {
            // Directly set innerHTML for HTML content.
            // The Content-Security-Policy meta tag should be part of the HTML string itself if needed,
            // or set via HTTP headers by the server.
            // For script.js, ensure `data` is trusted or sanitized if it can contain user-generated HTML with scripts.
            shadowRoot.innerHTML = '<meta http-equiv="Content-Security-Policy" content="default-src \'self\'; img-src \'self\' http: https: data:;">' + data; // Overwrite, don't append with += if clearing first
        } else if (selectedBodyVersion == 'plain-text') {
            const pre = document.createElement('pre');
            pre.textContent = data;
            shadowRoot.appendChild(pre);
        }
        // Process images after content is set
        processEmailImages(shadowRoot);
    }

    function updateEmailBody(selectedBodyVersion, emailId) {
        // Update the email body
        console.log('Updating email body ' + selectedBodyVersion + ' version for email ' + emailId);
        $.ajax({
            url: '/api/emails/' + emailId + '/body/' + selectedBodyVersion,
            type: 'GET',
            success: function (data) {
                // update the email body
                renderEmailBody(selectedBodyVersion, data);
            }
        });
    }

    function pickBestBodyVersion(bodyVersions) {
        // Pick the best body version
        console.log('Picking best body version');
        const preferredBodyVersions = ['html', 'watch-html', 'plain-text', 'raw'];
        for (var i = 0; i < preferredBodyVersions.length; i++) {
            if (bodyVersions.includes(preferredBodyVersions[i])) {
                return preferredBodyVersions[i];
            }
        }
        return 'raw';
    }

    const TAB_ICONS = {
        'html': 'bi-code-slash',
        'plain-text': 'bi-file-text',
        'raw': 'bi-file-binary',
        'watch-html': 'bi-smartwatch',
        'headers': 'bi-card-heading',
        'mime-tree': 'bi-diagram-3',
    };

    function updateEmailBodyVersions(bodyVersions, selectedBodyVersion, emailId) {
        const tabs = $('.email-body-tabs');
        tabs.empty();

        const allTabs = bodyVersions.concat(['headers', 'mime-tree']);
        for (let i = 0; i < allTabs.length; i++) {
            let tabName = allTabs[i];
            let testId = `email-body-version-tab-${tabName.toLowerCase().replace(' ', '-')}`;
            let icon = TAB_ICONS[tabName] || 'bi-file-earmark';
            let li = $('<li class="nav-item">');
            let link = $('<a class="nav-link">')
                .attr('data-testid', testId)
                .attr('role', 'tab')
                .html(`<i class="bi ${icon}"></i> ${tabName}`);

            if (tabName === selectedBodyVersion) {
                link.addClass('active');
            } else {
                link.click(function () {
                    updateEmailBodyVersions(bodyVersions, tabName, emailId);
                    if (tabName === 'headers') {
                        showRawHeaders(emailId);
                    } else if (tabName === 'mime-tree') {
                        showMimeTree(emailId);
                    } else {
                        hideExtraPanels();
                        updateEmailBody(tabName, emailId);
                    }
                });
            }
            li.append(link);
            tabs.append(li);
        }

        // Show/hide external images toggle only for HTML body versions
        const isHtmlTab = (selectedBodyVersion === 'html' || selectedBodyVersion === 'watch-html');
        if (isHtmlTab) {
            $('#externalImagesToggleContainer').show();
            $('.email-tab-toolbar').show();
        } else {
            $('#externalImagesToggleContainer').hide();
            $('.email-tab-toolbar').hide();
        }
    }

    function hideExtraPanels() {
        $('.email-raw-headers').hide();
        $('.email-mime-tree').hide();
        $('.email-content').show();
    }

    function showRawHeaders(emailId) {
        $('.email-content').hide();
        $('.email-mime-tree').hide();
        $('.email-tab-toolbar').hide();
        $.ajax({
            url: '/api/emails/' + emailId + '/headers',
            type: 'GET',
            success: function (data) {
                let text = '';
                // Sort headers for consistent display
                const keys = Object.keys(data).sort();
                for (const key of keys) {
                    for (const value of data[key]) {
                        text += key + ': ' + value + '\n';
                    }
                }
                $('.raw-headers-content').text(text);
                $('.email-raw-headers').show();
            }
        });
    }

    function showMimeTree(emailId) {
        $('.email-content').hide();
        $('.email-raw-headers').hide();
        $('.email-tab-toolbar').hide();
        $.ajax({
            url: '/api/emails/' + emailId + '/mime-tree',
            type: 'GET',
            success: function (data) {
                $('.mime-tree-content').empty().append(renderMimeTreeNode(data, emailId, 0));
                $('.email-mime-tree').show();
            }
        });
    }

    function renderMimeTreeNode(node, emailId, depth) {
        const container = $('<div>').addClass(depth > 0 ? 'mime-node' : '');

        // Header row: icon + type + details + actions
        const header = $('<div class="mime-node-header">');

        // Type badge
        const typeSpan = $('<span class="mime-node-type">').text(node.content_type);
        header.append(typeSpan);

        // Details
        const details = [];
        if (node.charset) details.push('charset=' + node.charset);
        if (node.encoding) details.push(node.encoding);
        if (node.size > 0 && !node.children) details.push(formatSize(node.size));
        if (node.content_id) details.push('CID: ' + node.content_id);

        if (details.length > 0) {
            header.append($('<span class="mime-node-details">').text(details.join(' · ')));
        }

        // Filename badge
        if (node.filename) {
            header.append($('<span class="badge bg-secondary">').text(node.filename));
        }

        // Disposition badge
        if (node.disposition) {
            const dispType = node.disposition.split(';')[0].trim();
            const badgeClass = dispType === 'attachment' ? 'bg-info' : 'bg-warning';
            header.append($('<span class="badge ' + badgeClass + '">').text(dispType));
        }

        // Actions (for leaf nodes with content)
        if (!node.children && node.size > 0) {
            const actions = $('<span class="mime-node-actions">');
            const ct = node.content_type.split(';')[0].trim();
            const isAttachment = node.disposition && node.disposition.startsWith('attachment');

            // "view" action — switch to the matching body tab for non-attachment body parts
            if (ct === 'text/html' && !isAttachment) {
                actions.append(
                    $('<a href="#" class="text-primary">').html('<i class="bi bi-eye"></i> view')
                        .attr('title', 'Switch to HTML tab')
                        .click(function(e) {
                            e.preventDefault();
                            const tab = $('[data-testid="email-body-version-tab-html"]');
                            if (tab.length) tab.click();
                        })
                );
            } else if (ct === 'text/plain' && !isAttachment) {
                actions.append(
                    $('<a href="#" class="text-primary">').html('<i class="bi bi-eye"></i> view')
                        .attr('title', 'Switch to plain-text tab')
                        .click(function(e) {
                            e.preventDefault();
                            const tab = $('[data-testid="email-body-version-tab-plain-text"]');
                            if (tab.length) tab.click();
                        })
                );
            } else if (ct === 'text/watch-html' && !isAttachment) {
                actions.append(
                    $('<a href="#" class="text-primary">').html('<i class="bi bi-eye"></i> view')
                        .attr('title', 'Switch to watch-html tab')
                        .click(function(e) {
                            e.preventDefault();
                            const tab = $('[data-testid="email-body-version-tab-watch-html"]');
                            if (tab.length) tab.click();
                        })
                );
            }

            // "preview" action for CID parts (images, etc.)
            if (node.content_id) {
                const cidUrl = '/api/emails/' + emailId + '/cid/' + node.content_id;
                actions.append(
                    $('<a href="#" class="text-secondary">').html('<i class="bi bi-eye"></i> preview')
                        .attr('title', 'Preview content')
                        .click(function(e) {
                            e.preventDefault();
                            showMimePreview(ct, cidUrl, node.filename || node.content_id);
                        })
                );
            }

            header.append(actions);
        }

        container.append(header);

        // Recursively render children
        if (node.children) {
            for (const child of node.children) {
                container.append(renderMimeTreeNode(child, emailId, depth + 1));
            }
        }
        return container;
    }

    function formatSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    function showMimePreview(contentType, url, title) {
        const body = $('#mimePreviewBody');
        body.empty();
        $('#mimePreviewModalLabel').text(title || 'Part Preview');
        $('#mimePreviewOpenLink').attr('href', url);

        if (contentType.startsWith('image/')) {
            body.append($('<img>').attr('src', url).css({'max-width': '100%', 'max-height': '70vh'}));
        } else if (contentType.startsWith('text/')) {
            // Fetch and display text content
            $.get(url, function(data) {
                body.append($('<pre>').text(typeof data === 'string' ? data : JSON.stringify(data, null, 2))
                    .css({'text-align': 'left', 'max-height': '60vh', 'overflow': 'auto', 'background': '#f8f9fa', 'padding': '12px', 'border-radius': '4px'}));
            });
        } else {
            body.append($('<p class="text-muted">').text('Preview not available for ' + contentType));
            body.append($('<a>').attr('href', url).attr('target', '_blank').text('Open in new tab'));
        }

        const modal = new bootstrap.Modal($('#mimePreviewModal')[0]);
        modal.show();
    }

    function updateEmailContent(email) {
        // Update the email content
        console.log('Updating email content ' + email.id);
        currentEmailId = email.id;
        updateEmailContentHeader(email);
        updateEmailAttachments(email.id);
        const selectedBodyVersion = pickBestBodyVersion(email.body_versions);
        updateEmailBodyVersions(email.body_versions, selectedBodyVersion, email.id);
        updateEmailBody(selectedBodyVersion, email.id);
    }

    function generateEmptyEmailListItem() {
        // display a nice message telling the user that there are no emails, centered and colspan on the complete row
        // with a warning icon
        return $('<tr class="email-item">').append($('<td colspan="6" class="text-center">')
                .append($('<i class="bi bi-exclamation-triangle icon">'))
                .append(' ')
                .append($('<span>').text('No emails found')));
    }

    function deleteEmail(emailId) {
        // Delete the email
        console.log('Deleting email');
        $.ajax({
            url: '/api/emails/' + emailId,
            type: 'DELETE',
            success: function (data) {
                // refresh the email list
                console.log('Email deleted');
                refreshEmailList();
                displayEmailList();
            },
            error: function (error) {
                console.log('Error deleting email');
                console.log(error);
            }
        });
    }

    function generateEmailListItem(email) {
        return $('<tr class="email-item">')
            .attr('data-testid', `email-row-${email.id}`)
            .append($('<td class="checkbox-col">').append(
                $('<input type="checkbox" class="form-check-input email-checkbox">')
                    .attr('data-email-id', email.id)
                    .attr('data-testid', 'email-checkbox-' + email.id)
                    .click(function(e) { e.stopPropagation(); })
            ))
            .append($('<td class="sender" data-testid="email-from-' + email.id + '">').append(formatEmailAddress(email.from)))
            .append($('<td class="preview" data-testid="email-preview-' + email.id + '">').append($('<strong>').text(email.subject + ' - ')).append($('<span>').css('font-style', 'italic').text(email.preview)))
            .append($('<td data-testid="email-attachment-icon-' + email.id + '">').append(email.has_attachments ? $('<i class="bi bi-paperclip icon">') : ''))
            .append($('<td class="date" data-testid="email-date-' + email.id + '">').text(formatDateTime(email.date)))
            .append($('<td data-testid="email-actions-' + email.id + '">')
                .append(
                    $('<i class="bi bi-trash icon" title="Delete" data-testid="email-delete-button-' + email.id + '">')
                    .click(function (event) {
                        event.stopPropagation();
                        console.log('Deleting email ' + email.id);
                        deleteEmail(email.id);
                    }))
                .append(
                    $('<i class="bi bi-envelope-arrow-up icon" title="Release..." data-testid="email-release-button-' + email.id + '">')
                    .click(function (event) {
                        // prevent cascade
                        event.stopPropagation();
                        console.log('Releasing email ' + email.id);
                        displayReleaseModal(email.id);
                    }))
            )
            .click(function () {
                console.log('Displaying email ' + email.id);
                updateEmailContent(email);
                displayEmailView();
            });
    }

    function generateMailboxListItem(mailbox) {
        return $('<li class="list-item">')
            .attr('data-testid', `mailbox-item-${mailbox.name}`)
            .text(mailbox.name)
            .attr('title', mailbox.name)
            .click(function () {
                var query = "mailbox:" + mailbox.name;
                // set current page to 1
                resetCurrentPage()
                updateSearchBoxAndRefreshEmailList(query);
            });
    }

    function updatePagination(pagination) {
        console.log('Updating pagination');
        lastKnownEmailCount = pagination.total_matches;
        $('#page-start').text(pagination.current_page);
        $('#page-total').text(pagination.total_pages);
        $('#total-matches').text(pagination.total_matches);
        if (pagination.is_first_page) {
            $('#prev-page').prop('disabled', true).css('cursor', 'default');
        } else {
            $('#prev-page').prop('disabled', false).css('cursor', 'pointer');
        }
        if (pagination.is_last_page) {
            $('#next-page').prop('disabled', true).css('cursor', 'default');
        } else {
            $('#next-page').prop('disabled', false).css('cursor', 'pointer');
        }
    }

    function resetCurrentPage() {
        $('#page-start').text(1);
    }

    function setSearchQuery(query) {
        $('.search-box input').val(query);
    }

    function formatDateTime(isoDateTime) {
        const dateObj = new Date(isoDateTime);
        return dateObj.toLocaleString('fr');
    }

    function formatEmailAddress(address) {
        if (address.name == null || address.name == '') {
            // put address in a span
            return $('<span>').text(address.address);
        } else {
            // put address in a span with a tooltip
            return $('<span>').attr('title', address.address).text(address.name);
        }
    }

    function formatEmailAddresses(addresses) {
        var formattedAddresses = $('<span>');
        for (var i = 0; i < addresses.length; i++) {
            var address = addresses[i];
            formattedAddresses.append(formatEmailAddress(address));
            if (i < addresses.length - 1) {
                formattedAddresses.append(', ');
            }
        }
        return formattedAddresses;
    }

    function showPopup(message, type = 'info') {
        // Define message types and Bootstrap 5 classes
        let popupClass = 'info';
        switch(type) {
            case 'success':
                popupClass = 'success';
                break;
            case 'warning':
                popupClass = 'warning';
                break;
            case 'error':
                popupClass = 'error';
                break;
        }

        // Create the popup HTML
        const popupHTML = `
            <div class="popup-message ${popupClass} alert alert-${type} alert-dismissible fade show">
                ${message}
                <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
        `;

        // Append the popup to the container
        const $popup = $(popupHTML).appendTo('#popup-container');

        // Automatically remove the popup after 5 seconds
        setTimeout(function() {
            $popup.fadeOut(function() {
                $(this).remove();
            });
        }, 5000);
    }
});

$(function () {
    const searchInput = $('.search-box input[type="text"]');
    const suggestionDisplay = $('#suggestion-display');

    // Add a document click listener to hide dropdown if clicked outside
    // $(document).on('click', function(event) { // OLD LOGIC - REMOVE
    //     // Check if the click target is not the search input and not part of the suggestions dropdown
    //     if (!$(event.target).is(searchInput) && !$(event.target).closest(suggestionsDropdown).length) { // OLD LOGIC - REMOVE
    //         clearAndHideDropdown(); // OLD LOGIC - REMOVE
    //     }
    // });

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

        // suggestionsDropdown.data('tokenStart', tokenStart); // OLD LOGIC - REMOVE
        // suggestionsDropdown.data('currentTokenLength', currentToken.length); // OLD LOGIC - REMOVE

        if (currentToken.trim() === '') {
            suggestionDisplay.text(''); // MODIFIED
            return;
        }

        $.ajax({
            url: `/api/filters/suggestions?term=${encodeURIComponent(currentToken)}`,
            type: 'GET',
            success: function (data) {
                // clearSuggestions(); // OLD LOGIC - REMOVE
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
        // Delay hiding to allow click on suggestion
        setTimeout(function() { suggestionDisplay.text(''); }, 150); // MODIFIED
    });

    // $(document).on('keydown', function (e) { // OLD LOGIC - REMOVE (Replaced by specific keydown on searchInput)
    //     if (e.key === "Escape") { // Modern browsers use "Escape"
    //         clearAndHideDropdown(); // OLD LOGIC - REMOVE
    //     }
    // });

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

    $('.bi-arrow-left').click(function () {
        displayEmailList();
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
                modal = $('#releaseEmailModal');
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

    $('[data-toggle="collapse"]').click(function () {
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
        // Delete all emails
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
                    $('#mailboxList').append(generateMaiboxListItem(mailbox));
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
        query = $('.search-box input').val();
        page = $('#page-start').text();
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
                emails = data.emails;
                emailList = $('.email-list .email-table tbody');
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
                attachments = data;
                attachmentList = $('.email-attachments');
                attachmentList.empty();
                if (attachments == null || attachments.length == 0) {
                    return;
                }
                attachmentsHtml = $('<span>');
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
        // Render the email body
        console.log('Rendering email body ' + selectedBodyVersion + ' version');
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
        preferedBodyVersions = ['html', 'watch-html', 'plain-text', 'raw'];
        for (var i = 0; i < preferedBodyVersions.length; i++) {
            if (bodyVersions.includes(preferedBodyVersions[i])) {
                return preferedBodyVersions[i];
            }
        }
        return 'raw';
    }

    function updateEmailBodyVersions(bodyVersions, selectedBodyVersion, emailId) {
        // Update the email body versions
        console.log('Updating email body versions');
        $('.email-body-versions').empty();
        $('.email-body-versions').append($('<strong>').text('Body versions: '));
        for (var i = 0; i < bodyVersions.length; i++) {
            let bodyVersion = bodyVersions[i];
            let testId = `email-body-version-tab-${bodyVersion.toLowerCase().replace(' ', '-')}`;
            if (bodyVersion == selectedBodyVersion) {
                $('.email-body-versions').append($('<span>').text(bodyVersion).css('font-weight', 'bold').attr('data-testid', testId));
            } else {
                $('.email-body-versions').append($('<span>').text(bodyVersion).attr('data-testid', testId).click(function () {
                    console.log('Switching to body version ' + bodyVersion);
                    updateEmailBodyVersions(bodyVersions, bodyVersion, emailId);
                    updateEmailBody(bodyVersion, emailId);
                }));
            }
            if (i < bodyVersions.length - 1) {
                $('.email-body-versions').append(', ');
            }
        }
    }

    function updateEmailContent(email) {
        // Update the email content
        console.log('Updating email content ' + email.id);
        updateEmailContentHeader(email);
        updateEmailAttachments(email.id);
        selectedBodyVersion = pickBestBodyVersion(email.body_versions);
        updateEmailBodyVersions(email.body_versions, selectedBodyVersion, email.id);
        updateEmailBody(selectedBodyVersion, email.id);
    }

    function generateEmptyEmailListItem() {
        // display a nice message telling the user that there are no emails, centered and colspan on the complete row
        // with a warning icon
        return $('<tr class="email-item">').append($('<td colspan="4">')
            .append($('<center>')
                .append($('<i class="bi bi-exclamation-triangle icon">'))
                .append(' ')
                .append($('<span>').text('No emails found'))));
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

    function generateMaiboxListItem(mailbox) {
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
        // Update the pagination
        console.log('Updating pagination');
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

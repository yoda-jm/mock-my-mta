$(function () {
    // initialize tooltips
    $('[title]').tooltip();
    // initialize the search
    setSearchQuery('');
    resetCurrentPage()
    refreshEmailList();

    $('.bi-arrow-left').click(function () {
        displayEmailList();
    });

    function displayEmailList() {
        $('.email-view').hide();
        $('.email-list').show();
    }

    function displayEmailView() {
        $('.email-list').hide();
        $('.email-view').show();
    }

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
                    var attachmentHtml = $('<span>').append($('<a>').attr('href', '/api/emails/' + emailId + '/attachments/' + attachment.id + '/content').text(attachment.filename));
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
        shadowRoot.innerHTML = '<meta http-equiv="Content-Security-Policy" content="default-src \'self\'">';
        if (selectedBodyVersion == 'raw') {
            shadowRoot.appendChild($('<pre>').text(data)[0]);
        } else if (selectedBodyVersion == 'html') {
            shadowRoot.innerHTML += data;
        } else if (selectedBodyVersion == 'plain-text') {
            shadowRoot.appendChild($('<pre>').text(data)[0]);
        } else if (selectedBodyVersion == 'watch-html') {
            shadowRoot.innerHTML += data;
        }
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
            if (bodyVersion == selectedBodyVersion) {
                $('.email-body-versions').append($('<span>').text(bodyVersion).css('font-weight', 'bold'));
            } else {
                $('.email-body-versions').append($('<span>').text(bodyVersion).click(function () {
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
            .append($('<td>').append(formatEmailAddress(email.from)))
            .append($('<td>').append($('<strong>').text(email.subject + ' - ')).append($('<span>').css('font-style', 'italic').text(email.preview)))
            .append($('<td>').text(formatDateTime(email.date)))
            .append($('<td>').append(
                $('<i class="bi bi-trash icon">'))
                .click(function () {
                    deleteEmail(email.id);
                })
            )
            .click(function () {
                console.log('Displaying email ' + email.id);
                updateEmailContent(email);
                displayEmailView();
            });
    }

    function generateMaiboxListItem(mailbox) {
        return $('<li class="list-item">')
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
});
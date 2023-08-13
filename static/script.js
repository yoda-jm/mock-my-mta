const pageSize = 5; // Number of items per page

let currentPage = 1;
let emailList = [];

function formatDateTime(isoDateTime) {
  const dateObj = new Date(isoDateTime);
  return dateObj.toLocaleString('fr');
}

function getBodyVersionsIcons(id, emailBodyVersions, clickable) {
  // body versions
  const bodyVersions = {
    'raw': 'fa-file-alt',
    'txt': 'fa-font',
    'html': 'fa-file-code',
    'watch-html': 'fa-clock',
    // Add your recognized body versions here
  };
  let bodyVersionsIcons = '';
  for (const [bodyVersion, icon] of Object.entries(bodyVersions)) {
    const hasBodyVersion = emailBodyVersions.includes(bodyVersion);
    if (clickable) {
      const iconClass = hasBodyVersion ? 'has-body-version fas ' + icon : 'fas ' + icon;
      const iconStyle = hasBodyVersion ? 'cursor: pointer' : 'color: lightgrey';
      bodyVersionsIcons += `<i class="${iconClass}" style="${iconStyle}" data-email-id="${id}" data-body-version="${bodyVersion}"></i> `;
    } else {
      const iconClass = hasBodyVersion ? 'fas ' + icon : 'fas ' + icon;
      const iconStyle = hasBodyVersion ? '' : 'color: lightgrey';
      bodyVersionsIcons += `<i class="${iconClass}" style="${iconStyle}"></i> `;
    }
  }
  return bodyVersionsIcons;
}

// Function to render the email table
function renderEmailTable() {
  const tableBody = $("#emailTable tbody");
  tableBody.empty();

  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;

  for (let i = startIndex; i < endIndex && i < emailList.length; i++) {
    const email = emailList[i];

    const attachmentIcon = email.has_attachment ? `<i class="fas fa-paperclip"></i>` : '';
    const bodyVersionsIcons = getBodyVersionsIcons(email.id, email.body_versions, false);
    const received_time = formatDateTime(email.received_time);

    tableBody.append(`
      <tr class="view-email" data-email-id="${email.id}">
        <td>${email.sender}</td>
        <td>${email.recipients.join(", ")}</td>
        <td>${email.subject}</td>
        <td>${received_time}</td>
        <td style="text-align: center">${attachmentIcon}</td>
	<td>${bodyVersionsIcons}</td>
	<td class="trash-icon"><i class="fas fa-trash"></i></td>
      </tr>
    `);
  }

  $("#currentPage").text(currentPage);

  // Add click event handlers for email view
  handleViewClick();
  handleDeleteClick()
}

// Function to update sorting icon in table header
function updateSortIcons(sortByField, sortByOrder) {
  $('.sortable').each(function () {
    const field = $(this).data('sortby');
    if (field === sortByField) {
      if (sortByOrder === 'asc') {
        $(this).html(`${field} <i class="fas fa-sort-up"></i>`);
      } else {
        $(this).html(`${field} <i class="fas fa-sort-down"></i>`);
      }
    } else {
      $(this).html(`${field} <i class="fas fa-sort"></i>`);
    }
  });
}

// Function to fetch email body content via AJAX
function fetchEmailBodyContent(emailId, bodyVersion) {
  const apiUrl = `/api/emails/${emailId}/body/${bodyVersion}`;
  $.ajax({
    url: apiUrl,
    type: "GET",
    success: function (data) {
      // Display the body content in the bodyContentDiv
      const bodyContentDiv = $("#bodyContent");
      // Create an iframe to display the HTML content securely
      const iframe = $('<iframe>', {
        sandbox: "allow-same-origin", // Allow same-origin content only (to keep it isolated)
        css: {
          width: "100%", // Set the width of the iframe
          height: "500px", // Set the height of the iframe (adjust as needed)
        },
      });
      bodyContentDiv.empty().append(iframe); // Replace existing content with the iframe

      let content = '';
      let contentType = 'text/html';
      if (bodyVersion === 'html') {
	content = data;
      } else {
        // Escape non-HTML content and wrap in a <pre> tag
        content = "<pre>" + $('<div/>').text(data).html() + "</pre>";
      }
      // Set the content of the iframe using Blob object
      const blob = new Blob([content], { type: contentType });
      blobUrl = URL.createObjectURL(blob);

      iframe.attr("src", blobUrl);
      $("#bodyContentDiv").show();
    },
    error: function (xhr, status, error) {
      console.error("Error fetching email body content:", error);
    },
  });
}

// Function to handle click event on body version icons
function handleBodyVersionClick() {
  $("i.has-body-version").on("click", function () {
    const emailId = $(this).data("email-id");
    const bodyVersion = $(this).data("body-version");
    fetchEmailBodyContent(emailId, bodyVersion);
  });
}

// Function to fetch email body content via AJAX
function fetchEmailAttachments(emailId, bodyVersion) {
  const apiUrl = `/api/emails/${emailId}/attachments`;
  $.ajax({
    url: apiUrl,
    type: "GET",
    success: function (data) {
      const attachmentsContentDiv = $("#attachmentsContent");
      attachmentsContentDiv.empty();
      for (let i = 0; i < data.length; i++) {
        const attachment = data[i];
        attachmentsContentDiv.append(`<p>${attachment.id} ${attachment.media_type} "${attachment.filename}"<p>`);
      }
      $("#attachmentsContentDiv").show();
    },
    error: function (xhr, status, error) {
      console.error("Error fetching email body content:", error);
    },
  });
}

function renderAttachments(id) {
  const apiUrl = `/api/emails/${id}/attachments`;
  $.ajax({
    url: apiUrl,
    type: "GET",
    success: function (attachments) {
      const attachmentsSpan = $("#attachments");
      attachmentsSpan.empty();
      if (attachments === null) {
        return;
      }
      for (let i = 0; i < attachments.length; i++) {
        const attachment = attachments[i];
        attachmentsSpan.append(`
	  <i>
	    <a href="/api/emails/${id}/attachments/${attachment.id}/content">
	      ${attachment.filename}
	    <a>
	    (${attachment.media_type}),
	  <i>
	`);
      }
    },
    error: function (xhr, status, error) {
      console.error("Error fetching email body content:", error);
    },
  });
}

function getDefaultBodyVersion(bodyVersions) {
  if (bodyVersions.includes("html")) {
    return "html";
  } else if (bodyVersions.includes("txt")) {
    return "txt";
  } else {
    return "raw";
  }
}

function renderEmail(email) {
  received_time = formatDateTime(email.received_time);
  const emailContentDiv = $("#emailContent");
  emailContentDiv.empty();
  emailContentDiv.append(`
    <p>Internal ID: <span id="currentEmail">${email.id}</span></p>
    <p>Sender: ${email.sender}</p>
    <p>Recipients: ${email.recipients.join(", ")}</p>
    <p>Date: ${received_time}</p>
    <p>Subject: ${email.subject}</p>
    <p>Attachments: <span id="attachments"></span></p>
    <div class="body-content" id="bodyContentDiv">
      <h3>Body Content</h3>
      <p>Body versions: <span id="bodyVersions"></span></p>
      <div id="bodyContent"></div>
    </div>
  `);

  // body versions
  bodyVersionsIcons = getBodyVersionsIcons(email.id, email.body_versions, true);
  $("#bodyVersions").html(bodyVersionsIcons);
  handleBodyVersionClick();

  // attachments
  renderAttachments(email.id);

  // render default body version
  fetchEmailBodyContent(email.id, getDefaultBodyVersion(email.body_versions));
  $("#emailContentDiv").show();
}

// Function to handle click event on an email line
function handleViewClick() {
  $("tr.view-email").on("click", function () {
    const line = $(this);
    const emailId = line.data("email-id");
    if ($("#currentEmail") !== null && $("#currentEmail").text() == emailId) {
    	return;
    }
    const table = line.parents().first();
    for(i = 0 ; i < table.children().length ; i++) {
      table.children().removeClass("selected-email");
    }
    line.addClass("selected-email");
    const apiUrl = `/api/emails/${emailId}`;
    $.getJSON(apiUrl, function (email) {
      renderEmail(email);
    });
  });
}

// Function to handle click event on delete
function handleDeleteClick() {
  $('.trash-icon').on('click', function (event) {
    // Prevent the default behavior (like following a link)
    event.preventDefault();
    // Prevent the event from propagating to parent elements
    event.stopPropagation();
  });

  // Click event for the trash icon
  $('.trash-icon i').on('click', function (event) {
    // Prevent the default behavior (like following a link)
    event.preventDefault();
    // Prevent the event from propagating to parent elements
    event.stopPropagation();

    const line = $(this).parents().first().parents().first();
    const emailId = line.data("email-id");
  
    const apiUrl = `/api/emails/${emailId}`;
    $.ajax({
      url: apiUrl,
      type: "DELETE",
      success: function (data) {
        console.log('deleted email ' + emailId);
        fetchEmails('date', 'desc');
      },
      error: function (xhr, status, error) {
        console.error("Error deleting email " + emailId + ":", error);
      },
    });
  });


}

// Function to handle pagination buttons
function handlePaginationButtons() {
  if (currentPage === 1) {
    $("#prevPage").addClass("disabled");
  } else {
    $("#prevPage").removeClass("disabled");
  }
  if (currentPage === Math.ceil(emailList.length / pageSize)) {
    $("#nextPage").addClass("disabled");
  } else {
    $("#nextPage").removeClass("disabled");
  }
}

// Function to fetch emails from the JSON API
function fetchEmails(sortByField, sortByOrder) {
  // Fetch emails with sorting criteria
  const apiUrl = `/api/emails?sort=${sortByField}&order=${sortByOrder}`;
  console.log("requesting: " + apiUrl);
  $.getJSON(apiUrl, function (data) {
    emailList = data;
    renderEmailTable();
    handlePaginationButtons();
    updateSortIcons(sortByField, sortByOrder);
  });
}

// Event listener for previous page button
$("#prevPage").on("click", function () {
  if (currentPage > 1) {
    currentPage--;
    renderEmailTable();
    handlePaginationButtons();
  }
});

// Event listener for next page button
$("#nextPage").on("click", function () {
  if (currentPage < Math.ceil(emailList.length / pageSize)) {
    currentPage++;
    renderEmailTable();
    handlePaginationButtons();
  }
});

// Event listener for sortable headers
$('.sortable').on('click', function () {
  const field = $(this).data('sortby');
  const hasSortUpClass = $(this).find('i').hasClass('fa-sort-up');
  const hasSortDownClass = $(this).find('i').hasClass('fa-sort-down');
  const hasSortClass = $(this).find('i').hasClass('fa-sort');
  if (hasSortUpClass) {
    fetchEmails(field, 'desc');
  } else if (hasSortDownClass) {
    fetchEmails(field, 'asc');
  } else if (hasSortClass) {
    if (field == 'date') {
      fetchEmails(field, 'desc');
    } else {
      fetchEmails(field, 'asc');
    }
  }
});

// Fetch the emails when the page loads
$(document).ready(function () {
  fetchEmails('date', 'desc');
});


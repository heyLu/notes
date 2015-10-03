var xhr = new XMLHttpRequest();
xhr.open('GET', '/tags');
xhr.setRequestHeader('Accept', 'application/json');
xhr.onload = function(ev) {
	window.tags = JSON.parse(xhr.responseText);
}
xhr.send();

var styleEl = document.createElement("style");
styleEl.textContent = `
.completions {
	position: absolute;
	left: -1000px;
	list-style-type: none;
	margin: 0;
	padding: 0;
	border: 1px solid #ddd;
	min-width: 10em;
}

.suggestion {
	padding-left: 5px;
}

.suggestion:hover {
	background-color: #eee;
}

.suggestion.selected {
	background-color: #ddd;
}
`;
document.head.appendChild(styleEl);

var completeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_";
function initCompletions(el) {
	var selectedIndex = 0;
	var selectedElement = null;

	var completionsEl = document.createElement("ul");
	completionsEl.classList.add("completions");
	document.body.appendChild(completionsEl);

	function resetCompletions() {
		selectedIndex = 0;
		selectedElement = null;
		completionsEl.innerHTML = "";
		completionsEl.style.left = "-1000px";
	}

	el.addEventListener("keydown", function(ev) {
		if (ev.key == "Tab") {
			ev.preventDefault();
		}
	});

	el.addEventListener("keyup", function(ev) {
		if (completeChars.indexOf(ev.key) == -1
				&& ev.key != "Backspace"
				&& ev.key != "Tab"
				&& ev.key != "ArrowUp"
				&& ev.key != "ArrowDown") {
			resetCompletions();
			return;
		}

		var rawTags = ev.target.value;
		if (ev.target.selectionStart != ev.target.selectionEnd) {
			return;
		}

		var tagStart = rawTags.lastIndexOf(" ", ev.target.selectionStart - 1);
		tagStart = tagStart == -1 ? 0 : tagStart + 1;
		var tagEnd = rawTags.indexOf(" ", ev.target.selectionStart - 1);
		tagEnd = tagEnd == -1 ? rawTags.length : tagEnd;

		var currentTag = rawTags.substring(tagStart, tagEnd);
		if (currentTag == "") {
			resetCompletions()
			return;
		}

		if (ev.key == "Tab") {
			ev.target.value = rawTags.substring(0, tagStart)
				+ selectedElement.textContent
				+ (ev.target.selectionStart == ev.target.value.length ? " " : "")
				+ rawTags.substring(tagEnd);
			resetCompletions();
			return;
		}

		if (ev.key == "ArrowUp") {
			selectedIndex = selectedIndex == 0 ? 0 : selectedIndex - 1;
		}
		if (ev.key == "ArrowDown") {
			selectedIndex += 1;
		}

		var completions = window.tags.
			filter((tag) => tag.startsWith(currentTag));

		completionsEl.style.left = ev.target.offsetLeft + "px";
		completionsEl.style.top = ev.target.offsetTop + 30 + "px";
		selectedElement = null;
		completionsEl.innerHTML = "";
		for (var i = 0; i < completions.length; i++) {
			var suggestionEl = document.createElement("li");
			suggestionEl.textContent = completions[i];
			suggestionEl.classList.add("suggestion");
			(function(tag) {
				suggestionEl.onclick = function() {
					ev.target.value = rawTags.substring(0, tagStart)
						+ tag
						+ (ev.target.selectionStart == ev.target.value.length ? " " : "")
						+ rawTags.substring(tagEnd);
					ev.target.focus();
					resetCompletions();
				}
			})(completions[i]);
			completionsEl.appendChild(suggestionEl);
		}

		if (completions.length > 0) {
			if (selectedElement != null) {
				selectedElement.classList.remove("selected");
			}
			selectedElement = completionsEl.children[selectedIndex];
			selectedElement.classList.add("selected");
		}
	});
}

initCompletions(document.querySelector("#tags"));

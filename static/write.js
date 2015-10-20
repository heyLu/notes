var editorEl = document.querySelector("#editor");
var codeMirror = new CodeMirror(editorEl, {
	lineWrapping: true,
	indentWithTabs: true,
	scrollbarStyle: "overlay",
	autofocus: true,
});

var savedName = null;
function getSavedName() {
	if (savedName != null) {
		return savedName;
	}

	var loc = new URL(location.href);
	if (!loc.searchParams.has('key')) {
		loc.searchParams.set('key', new Date().getTime());
	}
	savedName = `notes|${loc.toString()}`;
	history.replaceState({}, "notes", loc.toString());
	return savedName;
}

function saveDocument() {
	localStorage[getSavedName()] = codeMirror.getValue();
}

function getSavedDocument() {
	return localStorage[getSavedName()];
}

var needsSave = false;
function scheduleSave() {
	if (needsSave) {
		console.log("saving document");
		saveDocument();
		needsSave = false;
	}
	setTimeout(scheduleSave, 10000);
}
setTimeout(scheduleSave, 10000);

if (codeMirror.getValue() == "") {
	codeMirror.setValue(getSavedDocument() || "");
}

var titleEl = document.querySelector("#title");
var contentEl = document.querySelector("#content");

function updateTitle() {
	var firstLine = codeMirror.getLine(0);
	var title = "";
	if (firstLine.startsWith("# ")) {
		title = firstLine.substr(2);
	}

	document.title = `${title} - notes`;
	titleEl.value = title;
}

function updateDocument() {
	var firstLine = codeMirror.getLine(0);
	var start = {line: 0};
	if (firstLine.startsWith("# ")) {
		start = {line: 1};
	}
	contentEl.value = codeMirror.getRange(start, {line: codeMirror.lastLine()});
}

var wordsPerMinute = 250;
function getStats() {
	var numWords = codeMirror.getValue()
		.split(/\s/)
		.filter((word) => word.match(/\w/))
		.length;

	return {
		numWords: numWords,
		numCharacters: codeMirror.getValue()
			.split("")
			.filter((ch) => ch.match(/[^\s]/))
			.length,
		numMinutes: Math.ceil(numWords / wordsPerMinute),
	};
}

var wordStatsEl = document.querySelector("#stats-words");
var charStatsEl = document.querySelector("#stats-chars");
var timeStatsEl = document.querySelector("#stats-time");

function displayStats() {
	var stats = getStats();
	wordStatsEl.textContent = `${stats.numWords} ${stats.numWords == 1 ? "word" : "words"}`;
	charStatsEl.textContent = `${stats.numCharacters} ${stats.numCharacters == 1 ? "character" : "characters"}`;
	timeStatsEl.textContent = `${stats.numMinutes} ${stats.numMinutes == 1 ? "minute" : "minutes"}`;
}

displayStats();
updateTitle();
updateDocument();
codeMirror.on('change', function(cm, change) {
	displayStats();

	if (change.from.line == 0) {
		updateTitle();
	}
	updateDocument();

	needsSave = true;
});


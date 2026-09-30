(function () {
  "use strict";

  var MIN_CHARS = 3;
  var DEBOUNCE_MS = 250;

  var form = document.getElementById("city-search");
  var input = document.getElementById("city-input");
  var list = document.getElementById("suggestions");
  var button = document.getElementById("search-button");
  var cityField = document.getElementById("city-value");
  var countryField = document.getElementById("country-value");
  var hint = document.getElementById("search-hint");
  var errorBox = document.getElementById("search-error");

  if (!form || !input || !list) {
    return;
  }

  var timer = null;
  var sessionToken = null;
  var inFlight = null;

  function currentToken() {
    if (!sessionToken) {
      sessionToken = newToken();
    }
    return sessionToken;
  }

  function newToken() {
    if (window.crypto && window.crypto.randomUUID) {
      return window.crypto.randomUUID();
    }
    return "s-" + Date.now() + "-" + Math.random().toString(16).slice(2);
  }

  function showError(message) {
    errorBox.textContent = message;
    errorBox.hidden = !message;
  }

  function clearSelection() {
    cityField.value = "";
    countryField.value = "";
    button.disabled = true;
  }

  function closeList() {
    list.innerHTML = "";
    list.hidden = true;
    input.setAttribute("aria-expanded", "false");
  }

  function renderSuggestions(items) {
    list.innerHTML = "";

    if (!items.length) {
      var empty = document.createElement("li");
      empty.className = "suggestion is-empty";
      empty.textContent = "No matching cities.";
      list.appendChild(empty);
      list.hidden = false;
      input.setAttribute("aria-expanded", "true");
      return;
    }

    items.forEach(function (item) {
      var li = document.createElement("li");
      li.className = "suggestion";
      li.setAttribute("role", "option");
      li.tabIndex = 0;

      var main = document.createElement("span");
      main.className = "suggestion-main";
      main.textContent = item.mainText || item.text;
      li.appendChild(main);

      if (item.secondaryText) {
        var sub = document.createElement("span");
        sub.className = "suggestion-sub";
        sub.textContent = item.secondaryText;
        li.appendChild(sub);
      }

      function choose() {
        selectPlace(item);
      }
      li.addEventListener("click", choose);
      li.addEventListener("keydown", function (e) {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          choose();
        }
      });

      list.appendChild(li);
    });

    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
  }

  function fetchSuggestions(value) {
    if (inFlight) {
      inFlight.abort();
    }
    inFlight = new AbortController();

    var url = "/api/locations/autocomplete?input=" + encodeURIComponent(value) +
              "&sessionToken=" + encodeURIComponent(currentToken());

    fetch(url, { signal: inFlight.signal, headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) {
          throw new Error("lookup failed");
        }
        return res.json();
      })
      .then(function (data) {
        showError("");
        renderSuggestions(data.suggestions || []);
      })
      .catch(function (err) {
        if (err.name === "AbortError") {
          return;
        }
        closeList();
        showError("City search is unavailable right now. Please try again.");
      });
  }

  function selectPlace(item) {
    var url = "/api/locations/" + encodeURIComponent(item.placeId) +
              "?sessionToken=" + encodeURIComponent(currentToken());

    fetch(url, { headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) {
          throw new Error("details failed");
        }
        return res.json();
      })
      .then(function (city) {
        if (!city.city || !city.countryCode) {
          throw new Error("incomplete city");
        }
        input.value = item.mainText || item.text;
        cityField.value = city.city;
        countryField.value = city.countryCode;
        button.disabled = false;
        hint.textContent = "Ready to explore " + city.city + ", " + city.countryCode + ".";
        showError("");
        closeList();

        sessionToken = null;
      })
      .catch(function () {
        clearSelection();
        showError("That city could not be confirmed. Please pick another.");
      });
  }

  input.addEventListener("input", function () {

    clearSelection();
    hint.textContent = "Type at least 3 characters and select a suggestion.";

    var value = input.value.trim();
    window.clearTimeout(timer);

    if (value.length < MIN_CHARS) {
      closeList();
      showError("");
      return;
    }

    timer = window.setTimeout(function () {
      fetchSuggestions(value);
    }, DEBOUNCE_MS);
  });

  input.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      closeList();
    }
  });

  document.addEventListener("click", function (e) {
    if (!form.contains(e.target)) {
      closeList();
    }
  });

  form.addEventListener("submit", function (e) {
    if (!cityField.value || !countryField.value) {
      e.preventDefault();
      showError("Please select a city from the suggestions.");
    }
  });

  clearSelection();
})();
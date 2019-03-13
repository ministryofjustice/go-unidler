{{define "javascript"}}
(function () {
  // Delay before redirecting to unidled app
  var DELAY = 5000;
  var url = '/events/';
  var message = document.getElementById("message");

  var urlparams = new URLSearchParams(window.location.search);
  var host = urlparams.get("host");
  if (host) {
    url += '?host=' + host;
  }
  var source = new EventSource(url);

  function redirect() {
    window.location.href = "https://{{.}}/";
  }

  function showMessage(msg) {
    message.innerHTML = msg;
  }

  function showFinalState(finalState, finalMessage) {
    source.close();

    var elem = document.getElementById(finalState);
    elem.classList.remove("hidden");

    showMessage(finalMessage);
  }

  source.onmessage = function(e) {
    showMessage(e.data);
  };

  source.onerror = function (e) {
    showFinalState("failure", e.data);
  };

  source.addEventListener("success", function (e) {
    showFinalState("success", e.data);
    window.setTimeout(redirect, DELAY);
  }, false);
})();
{{end}}

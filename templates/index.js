{{define "javascript"}}
(function () {
  // Delay before redirecting to unidled app
  var DELAY = 5000;
  var url = '/events/';
  var status = document.getElementById("status");
  var urlparams = new URLSearchParams(window.location.search);
  var host = urlparams.get("host");
  if (host) {
    url += '?host=' + host;
  }
  var source = new EventSource(url);

  function redirect() {
    window.location.href = "https://{{.}}/";
  }

  function updateStatus(msg) {
    status.innerHTML = msg;
  }

  source.onmessage = function(e) {
    updateStatus(e.data);
  };

  source.onerror = function (e) {
    updateStatus(e.data);
    var msg  = document.getElementsByClassName("failure")[0];
    msg.classList.remove("hidden");
  };

  source.addEventListener("success", function (e) {
    updateStatus(e.data);
    window.setTimeout(redirect, DELAY);
    var msg = document.getElementsByClassName("success")[0];
    msg.classList.remove("hidden");
  }, false);
})();
{{end}}

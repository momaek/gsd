(function () {
  'use strict';

  function initSidebar() {
    var pathname = window.location.pathname.replace(/\/+$/, "");
    var current = $(".sphinxsidebar ul a").filter(function (index, a) {
      return pathname === a.pathname;
    });
    current.addClass("current");
    current.parents(".collapse").addClass("show");
    current.parents("li").addClass("opend");
    current.parent().next(".collapse").addClass("show");
  }

  $(document).ready(function () {
    initSidebar();
  });

})();

(function () {
  'use strict';

  function initSidebar() {
    var pathname = window.location.pathname.replace(/\/+$/, "");
    var current = $(".sphinxsidebar ul a").filter(function (index, a) {
      return pathname === a.pathname;
    });
    current.addClass("current");
    var ul = current.parents(".collapse").addClass("show");
    ul.prev().find('[data-toggle="collapse"]').removeClass("collapsed");

    current.parent().next(".collapse").addClass("show");
  }

  $(document).ready(function () {
    initSidebar();
  });

  // bootstrap
  $('[data-toggle="tooltip"]').tooltip()

  $(document).on("click", "#btn-printer", function () {
    window.print();
  })

})();

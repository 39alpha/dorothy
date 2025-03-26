const checkAvailability = (entity, route, options = {}) => {
  let { src, dst, startDisabled, original, trigger, target_parent } = Object.assign({
    src: "#name",
    dst: "#availability",
  }, options);

  const disable = () => {
    if (trigger) {
      $(trigger).prop("disabled", true);
    }
  };

  const enable = () => {
    if (trigger) {
      $(trigger).prop("disabled", false);
    }
  };

  const addClass = (cls) => {
    if (target_parent) {
      $(dst).parent().addClass(cls);
    } else {
      $(dst).addClass(cls);
    }
  };

  const removeClass = (cls) => {
    if (target_parent) {
      $(dst).parent().removeClass(cls);
    } else {
      $(dst).removeClass(cls);
    }
  };

  if (startDisabled) {
    disable();
  }

  let timeout = undefined;

  $(src).on("keyup", () => {
    clearTimeout(timeout);

    if ($(src).val() == "") {
      disable();

      addClass("hidden");
      return;
    } else if ($(src).val() == original) {
      enable();

      addClass("hidden");
      return;
    }

    timeout = setTimeout(() => {
      fetch(route, {
        method: "POST",
        headers: {
          accept: "application/json",
          ["Content-Type"]: "application/json",
        },
        body: JSON.stringify({ name: $(src).val() }),
      }).then(async (response) => {
        try {
          return { response, body: await response.json() };
        } catch (_err) {
          return { response, body: {} };
        }
      }).then(({ response, body }) => {
        if (!response.ok) {
          disable();


          addClass("text-red-500")
          removeClass("text-green-500 text-blue-500 hidden")
          $(dst).html(
              '<i class="fa-solid fa-times mr-2"></i>' + body.error ??
                "cannot check for availability right now",
            );
        } else if (body.needsRewrite) {
          enable();

          addClass("font-bold text-blue-500")
          removeClass("text-red-500 text-green-500 hidden")
          $(dst)
            .html(
              `<i class="fa-solid fa-check mr-2"></i>Your ${entity} will be named "${body.rewrittenName}"`,
            );
        } else if (body.message) {
          enable();

          addClass("font-bold text-green-500")
          removeClass("text-red-500 text-blue-500 hidden")
          $(dst)
            .html('<i class="fa-solid fa-check mr-2"></i>' + body.message);
        }
      });
    }, 500);
  });
};

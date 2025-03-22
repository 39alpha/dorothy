const checkAvailability = (entity, route, options = {}) => {
  const { src, dst, startDisabled, original, trigger } = Object.assign({
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

  if (startDisabled) {
    console.log("disabling");
    disable();
  }

  let timeout = undefined;

  $(src).on("keyup", () => {
    clearTimeout(timeout);

    if ($(src).val() == "") {
      disable();

      $(dst).addClass("hidden");
      return;
    } else if ($(src).val() == original) {
      enable();

      $(dst).addClass("hidden");
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

          $(dst)
            .addClass("error")
            .removeClass("hidden")
            .html(body.error ?? "cannot check for availability right now");
        } else if (body.needsRewrite) {
          enable();

          $(dst)
            .removeClass("error hidden")
            .html(`Your ${entity} will be created as "${body.rewrittenName}".`);
        } else if (body.message) {
          enable();

          $(dst)
            .removeClass("error hidden")
            .html(body.message);
        }
      });
    }, 500);
  });
};

const checkAvailability = (entity, route, options = {}) => {
    const { src, dst, startDisabled, original, trigger, target_parent } = Object
        .assign({
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

                    addClass("text-red-500");
                    removeClass("text-green-500 text-blue-500 hidden");
                    $(dst).html(
                        '<i class="fa-solid fa-times mr-2"></i>' + body.error ??
                            "cannot check for availability right now",
                    );
                } else if (body.needsRewrite) {
                    enable();

                    addClass("font-bold text-blue-500");
                    removeClass("text-red-500 text-green-500 hidden");
                    $(dst)
                        .html(
                            `<i class="fa-solid fa-check mr-2"></i>Your ${entity} will be named "${body.rewrittenName}"`,
                        );
                } else if (body.message) {
                    enable();

                    addClass("font-bold text-green-500");
                    removeClass("text-red-500 text-blue-500 hidden");
                    $(dst)
                        .html(
                            '<i class="fa-solid fa-check mr-2"></i>' +
                                body.message,
                        );
                }
            });
        }, 500);
    });
};

const request_confirmation = (event) => {
    $(event.target).parents().find("dialog").get(0).showModal();
};

const getParentForm = (entity) => $(entity).is('form') ? $(entity) : $($(entity).parents('form').get(0));

const findDialogAndApply = (foo, entity) => {
    let dialog = undefined;

    if (entity) {
        if ($(entity).is('dialog')) {
            dialog = $(entity).get(0);
        } else {
            dialog = $(entity).parents('dialog').get(0);
        }

        if (dialog) {
            foo(dialog);
        }
    }
}

const close_dialog = (entity) => findDialogAndApply((dialog) => dialog.close(), entity);
const show_dialog = (entity) => findDialogAndApply((dialog) => dialog.showModal(), entity);

const send_delete = (resource, redirect, event) => {
    event.preventDefault();

    fetch(resource, {
        method: "DELETE",
        headers: {
            accept: "application/json",
        },
    }).then(async (response) => {
        try {
            return { response, body: await response.json() };
        } catch (_) {
            return {
                response,
                body: {
                    error: "The server returned an unexpected response",
                },
            };
        }
    }).then(({ response, body }) => {
        if (!response.ok || body.error) {
            throw new Error(
                body.error ?? "An unexpected error occurred; try again later",
            );
        }

        window.location.href = redirect;
    }).catch((err) => {
        const message_div = $(event.target).find(".error").removeClass("hidden")
            .html(err);
        $("html, body").animate({
            scrollTop: message_div.offset().top,
        });
    });
};

const confirm_name_change = (entity) => {
    const form = getParentForm(entity);

    if (form) {
        const dataset_name = form.find("input[name='name']");
        const was = dataset_name.attr("data-value");
        const now = dataset_name.prop("value");
        const dialog = $("#name_change_confirmation")

        if (now != was) {
            $("#name_was").html(was);
            $("#name_now").html(now);
            show_dialog(dialog);
        } else if (confirm_name_change.next) {
            confirm_name_change.next(form, dialog);
        }
    }

    return false;
};

const confirm_visibility_change = (entity) => {
    const form = getParentForm(entity);

    if (form) {
        const is_private = form.find("input[name='isPrivate']");
        const was = is_private.attr("data-value") == "true" ? "private" : "public";
        const now = is_private.prop("checked") ? "private" : "public";
        const dialog = $("#visibility_change_confirmation");

        if (now != was) {
            $("#visibility_was").html(was);
            $("#visibility_now").html(now);

            const warning = (now == "public")
                ? "Anyone will be able to view the dataset, but only users with admin or write access will be able to modify it."
                : "Only users with read, write or admin access to the respository will be able to view the dataset.";
            $("#visibility_warning").html(warning);
            show_dialog(dialog);
        } else if (confirm_visibility_change.next) {
            confirm_visibility_change.next(form, dialog);
        }
    }

    return false;
};

confirm_name_change.next = (entity, dialog) => {
    if (!dialog) {
        close_dialog(entity);
    }
    confirm_visibility_change(getParentForm(entity));
};

confirm_visibility_change.next = (entity, dialog) => {
    if (!dialog) {
        close_dialog(entity);
    }

    getParentForm(entity).get(0).submit();
};

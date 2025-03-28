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

const requestConfirmation = (event) => {
    const dialog = $(event.target).parents("form").find("dialog").get(0);
    dialog.showModal();
};

const getParentForm = (entity) =>
    $(entity).is("form") ? $(entity) : $($(entity).parents("form").get(0));

const findDialogAndApply = (foo, entity) => {
    let dialog = undefined;

    if (entity) {
        if ($(entity).is("dialog")) {
            dialog = $(entity).get(0);
        } else {
            dialog = $(entity).parents("dialog").get(0);
        }

        if (dialog) {
            foo(dialog);
        }
    }
};

const closeDialog = (entity) =>
    findDialogAndApply((dialog) => dialog.close(), entity);
const showDialog = (entity) =>
    findDialogAndApply((dialog) => dialog.showModal(), entity);
const close_all_dialogs = () => $("dialog").each((_, d) => closeDialog(d));

const readResponse = async (response) => {
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
};

const guardResponse = async ({ response, body }) => {
    if (!response.ok || body?.error) {
        console.log(body);
        throw new Error(
            body?.error ?? "An unexpected error occurred; try again later",
        );
    }
    return { response, body };
};

const sendDelete = (resource, redirect, event) => {
    event.preventDefault();

    close_all_dialogs();

    fetch(resource, {
        method: "DELETE",
        headers: {
            accept: "application/json",
        },
    }).then(readResponse).then(guardResponse).then(() => {
        window.location.href = redirect;
    }).catch((err) => {
        const message_div = $(event.target).find(".error").removeClass("hidden")
            .html(err);
        $("html, body").animate({
            scrollTop: message_div.offset().top,
        });
    });
};

const confirmNameChange = (entity) => {
    const form = getParentForm(entity);

    if (form) {
        const dataset_name = form.find("input[name='name']");
        const was = dataset_name.attr("data-value");
        const now = dataset_name.prop("value");
        const dialog = $("#name_change_confirmation");

        if (now != was) {
            $("#name_was").html(was);
            $("#name_now").html(now);
            showDialog(dialog);
        } else if (confirmNameChange.next) {
            confirmNameChange.next(form, dialog);
        }
    }

    return false;
};

const confirmVisibilityChange = (entity) => {
    const form = getParentForm(entity);

    if (form) {
        const is_private = form.find("input[name='isPrivate']");
        const was = is_private.attr("data-value") == "true"
            ? "private"
            : "public";
        const now = is_private.prop("checked") ? "private" : "public";
        const dialog = $("#visibility_change_confirmation");

        if (now != was) {
            $("#visibility_was").html(was);
            $("#visibility_now").html(now);

            const warning = (now == "public")
                ? "Anyone will be able to view the dataset, but only users with admin or write access will be able to modify it."
                : "Only users with read, write or admin access to the respository will be able to view the dataset.";
            $("#visibility_warning").html(warning);
            showDialog(dialog);
        } else if (confirmVisibilityChange.next) {
            confirmVisibilityChange.next(form, dialog);
        }
    }

    return false;
};

confirmNameChange.next = (entity, dialog) => {
    if (!dialog) {
        closeDialog(entity);
    }
    confirmVisibilityChange(getParentForm(entity));
};

confirmVisibilityChange.next = (entity, dialog) => {
    if (!dialog) {
        closeDialog(entity);
    }

    getParentForm(entity).get(0).submit();
};

const confirmUpdates = (event) => {
    return confirmNameChange($('form[name="update"]'));
};

const searchUsers = (pattern, limit) => {
    let params = [];

    if (pattern != undefined) {
        params.push(`q=${pattern}`);
    }
    if (limit != undefined) {
        params.push(`limit=${limit}`);
    }

    return fetch(`/user/search?${params.join("&")}`, {
        method: "GET",
        headers: {
            accept: "application/json",
        },
    }).then((response) => response.json()).catch((err) => ({ error: err }));
};

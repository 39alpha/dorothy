const checkPasswordConfirmation = (elements = {}) => {
    elements = Object.assign({
        password: "#password",
        confirm: "#confirmPassword",
        check: "#passwordCheck",
    }, elements);

    const password = $(elements.password).val();
    const confirmPassword = $(elements.confirm).val();

    const flash = $(elements.check);
    const submit = elements.submit
        ? $(elements.submit)
        : $(elements.password).parent().find('button[type="submit"]');
    flash.removeClass("text-red-500 text-green-500");
    if (password != confirmPassword) {
        flash.addClass("text-red-500").html("Passwords do not match!");
        submit.prop("disabled", true);
    } else {
        flash.addClass("text-green-500").html("Passwords match.");
        submit.prop("disabled", false);
    }
    flash.parent().removeClass("hidden");
};

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
    console.log(body);
    if (!response.ok || body?.error) {
        throw new Error(
            body?.error ?? "An unexpected error occurred; try again later",
        );
    }
    return { response, body };
};

const displayError = (target, err) => {
    console.error(err);
    const message_div = $(target)
        .find(".error")
        .html(err.message ?? err)
        .parent()
        .removeClass("hidden");
    $("html, body").animate({
        scrollTop: message_div.offset().top,
    });
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
    }).catch((err) => displayError(event.target, err));
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

const getAddedUserPrivileges = (event) => {
    const users = [];
    $(event.target).parents("form").find("fieldset")
        .each((_, e) => {
            users.push({
                index: $(e).attr("data-index"),
                user_id: $(e).attr("data-user-id"),
                name: $(e).attr("data-name"),
                email: $(e).attr("data-email"),
                privilege_code: $(e).attr("data-privilege-code"),
            });
        });
    return users;
};

const addUserPrivilege = (element) => {
    element = $(element);

    const index = 1 + getAddedUserPrivileges(element)
        .map((fieldset) => fieldset.index)
        .reduce((a, b) => Math.max(a, b), 0);

    const user_id = $(element).attr("data-user-id");
    const name = $(element).attr("data-name");
    const email = $(element).attr("data-email");

    const privilege = $($("#new-privilege").prop("content")).find("fieldset")
        .clone();

    $(privilege).attr("data-index", index);
    $(privilege).attr("data-user-id", user_id);
    $(privilege).attr("data-name", name);
    $(privilege).attr("data-email", email);

    $(privilege).find(".name").html(name);
    $(privilege).find(".email").html(email);
    $(privilege).find('input[type="radio"]').each((_, r) => {
        const v = $(r).attr("value");
        $(r).attr("name", `privilege-${index}`);
        $(r).attr("id", `privilege-${v}-${index}`);
    });
    $(privilege).find("label").each((_, l) => {
        const v = $(l).text();
        $(l).attr("for", `privilege-${v}-${index}`);
    });
    $(privilege).find("checkbox").attr("name", `remove-privilege-${index}`);
    $(privilege).find('label[for="remove-privilege"]').attr(
        "for",
        `remove-privilege-${index}`,
    );

    $("#privileges").prepend(privilege);

    $("#search").val("");
    $("#search-results").empty().addClass("hidden");
    element.remove();
};

const searchUsersForPrivileges = (element, target, duration = 500) => {
    let timeout = undefined;
    $(element).on("keyup", (event) => {
        clearTimeout(timeout);

        const value = $(event.target).val();

        if (!value || value.length < 3) {
            $(target).empty();
            $(target).addClass("hidden");
            return;
        }

        timeout = setTimeout(() => {
            const existing_users = new Set(
                getAddedUserPrivileges({ target: element }).map((user) =>
                    user.email
                ),
            );
            searchUsers(value).then(({ users }) => {
                $(target).empty();
                if (users && users.length != 0) {
                    const user_list = $($("#found-users").prop("content")).find(
                        "ul",
                    ).clone();
                    users.forEach((user) => {
                        if (!existing_users.has(user.email)) {
                            const u = $($("#user").prop("content")).clone();
                            const slots = u.find("slot");
                            $(u).find("button").attr("data-user-id", user.id);
                            $(u).find("button").attr("data-name", user.name);
                            $(u).find("button").attr("data-email", user.email);
                            $(slots.get(0)).html(user.name);
                            $(slots.get(1)).html(user.email);
                            user_list.append(u);
                        }
                    });
                    $(target).append(user_list);
                } else {
                    $(target).append(
                        $($("#no-users-found").prop("content")).clone(),
                    );
                }
                $(target).removeClass("hidden");
            }).catch((err) => console.error(err));
        }, duration);
    });
};

const updatePrivileges = async (resource, entity) => {
    const form = $($(entity).parents("form").get(0));
    const fieldset = $($(entity).parents("fieldset").get(0));

    const asset_id = parseInt(form.find('input[name="id"]').val());
    const user_id = parseInt(fieldset.attr("data-user-id"));
    const privilege_code = fieldset.attr("data-privilege-code");
    const checked = fieldset.find('input[type="radio"]:checked');

    if (checked.length === 1) {
        if (privilege_code == checked.val()) {
            form.find(".error").parent().addClass("hidden");
        } else {
            const payload = {
                id: asset_id,
                userId: user_id,
                privilegeCode: checked.val(),
            };

            fetch(resource, {
                method: "POST",
                headers: {
                    accept: "application/json",
                    ["Content-Type"]: "application/json",
                },
                body: JSON.stringify(payload),
            }).then(readResponse).then(guardResponse).then(() => {
                form.find(".error").parent().addClass("hidden");
            }).catch((err) => displayError(form, err));
        }
    }
};

const removePrivileges = (resource, entity) => {
    const form = $($(entity).parents("form").get(0));
    const fieldset = $($(entity).parents("fieldset").get(0));

    if (fieldset.find('input[type="radio"]:checked').length == 0) {
        $(entity).parents("fieldset").remove();
        return;
    }

    const payload = {
        id: parseInt(form.find('input[name="id"]').val()),
        userId: parseInt(fieldset.attr("data-user-id")),
    };

    fetch(resource, {
        method: "DELETE",
        headers: {
            accept: "application/json",
            ["Content-Type"]: "application/json",
        },
        body: JSON.stringify(payload),
    }).then(readResponse).then(guardResponse).then(() => {
        form.find(".error").parent().addClass("hidden");
        $(entity).parents("fieldset").remove();
    }).catch((err) => displayError(form, err));
};

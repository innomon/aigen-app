import {fetchMe} from "../services/userService.js";

let user; 
let currentUserPromise;

export function getUser() {
    return user;
}

export function ensureUser(){
    if (user) {
        return user;
    }

    const proceed = confirm("You must log in to perform this action. Do you want to log in now?");
    if (proceed) {
        window.location.href = "/portal?ref=" + encodeURIComponent(window.location.href);
    }
    return false; 
}

export function getActiveRoleId() {
    return localStorage.getItem('activeRoleId') || (user ? String(user.defaultRoleId || user.rolesDetails?.[0]?.id || '') : null);
}

export function setActiveRoleId(roleId) {
    localStorage.setItem('activeRoleId', roleId);
    window.location.reload();
}

export function getActiveRole() {
    if (!user || !user.rolesDetails) return null;
    const activeId = getActiveRoleId();
    return user.rolesDetails.find(r => String(r.id) === activeId) || user.rolesDetails[0];
}

//single flight
export async function fetchUser() {
    if (user) return;
    if (currentUserPromise) {
        return currentUserPromise;
    }

    try {
        currentUserPromise = fetchMe();
        user = await currentUserPromise;
        if (user && !localStorage.getItem('activeRoleId') && user.defaultRoleId) {
            localStorage.setItem('activeRoleId', user.defaultRoleId);
        }
        return user;
    } catch (error) {
        return false;
    } finally {
        // Clear the promise after completion
        currentUserPromise = null;
    }
}
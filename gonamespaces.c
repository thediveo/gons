/*
 * Initializer function to join this(!) process to specific Linux-kernel
 * namespaces before the Go runtime spins up and blocks joining certain
 * namespaces, especially mount namespaces due to creating multiple OS
 * threads.
 *
 * Compared to libcontainer's nsenter Go package, we switch namespaces on our
 * own process (to the extend this is possible), and we switch into existing
 * namespaces. In contrast, libcontainer's nsenter creates new namespaces for
 * child processes it creates, so it's a completely different usecase.
 *
 * The namespaces to switch into are passed to us via environment variables.
 * They're in the form of "netns=/proc/self/ns/net". Please take note that the
 * names of the env vars are namespace names, with "ns" appended to avoid name
 * conflicts with common environment variable names such as "pid", et cetera.
 *
 * Copyright 2019 Harald Albrecht.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.You may obtain a copy
 * of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

/* Fun stuff... */
#define _GNU_SOURCE
#include <sched.h>
#include <unistd.h>
#include <sys/syscall.h>

/* Booooring stuff... */
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <fcntl.h>
#include <stdarg.h>
#include <errno.h>
#include <limits.h>

/* Describes a specific type of Linux kernel namespace supported by gons. */
struct ns_t {
    char *symname; /* namespace symbolic name, such as "mnt", "net", et cetera. */
    int nstype; /* CLONE_NEWxxx constant for this type of namespace. */
};

/*
 * Defines the list of supported namespaces which can be set by gons before
 * the Go runtime spins up. Please note that setting the PID namespace will
 * never apply to us, but only to our children.
 */
static const struct ns_t namespaces[] = {
    { "cgroupns", CLONE_NEWCGROUP },
    { "ipcns", CLONE_NEWIPC },
    { "mntns", CLONE_NEWNS },
    { "netns", CLONE_NEWNET },
    { "pidns", CLONE_NEWPID },
    { "userns", CLONE_NEWUSER },
    { "utsns", CLONE_NEWUTS }
};

/*
 * If not NULL, then points to a buffer with an error message for later
 * consumption by an application in order to detect namespace switching
 * errors. Its size not only accounts for the maximum path size, but also for
 * some descriptive text prefixing it.
 */
char *gonsmsg;
static unsigned int maxmsgsize;

/*
 * Our last-resort error reporting, which the application should later pick up
 * by calling the Go function gons.Status().
 */
static void logerr(const char *format, ...) {
    va_list args;
    /* Create a buffer if not done so. */
    if (!gonsmsg) {
        maxmsgsize = 256 + PATH_MAX;
        gonsmsg = (char *) malloc(maxmsgsize);
        if (!gonsmsg) {
            /* Handle oom and protect against overwriting this message */
            gonsmsg = "malloc error";
            maxmsgsize = 0;
            return;
        }
    }
    /* Generate the error message... */
    va_start(args, format);
    /*
     * He who has never ignored printf()'s return value, cast the first stone.
     */
    vsnprintf(gonsmsg, maxmsgsize, format, args);
    va_end(args);
}

/*
 * Switch into the Linux kernel namespaces specified through env variables:
 * these env vars reference namespaces in the filesystem, such as
 * "netns=/proc/$PID/ns/net". See the static constant "namespaces" above for
 * the set of Linux namespaces supported.
 */
void gonamespaces(void) {
    for (int nsidx = 0; nsidx < sizeof(namespaces) / sizeof(namespaces[0]); ++nsidx) {
        char *nsenv = getenv(namespaces[nsidx].symname);
        if (nsenv != NULL && *nsenv != '\0') {
            /*
             * There's an env var specified for this namespace, and it should
             * reference a namespace of this type in the filesystem...
             */
            int nsref = open(nsenv, O_RDONLY);
            if (nsref < 0) {
                logerr("package gons: invalid %s reference \"%s\": %s", 
                    namespaces[nsidx].symname, nsenv,
                    strerror(errno));
                return;
            }
            /*
            * Do not use the glibc version of setns, but go for the syscall
            * itself. This allows us to avoid dynamically linking to glibc
            * even when using cgo, resorting to musl, et cetera. As musl is a
            * mixed bag in terms of its glibc compatibility, especially in
            * such dark corners as Linux namespaces, we try to minimize
            * problematic dependencies here.
            *
            * A useful reference is Dominik Honnef's blog post "Statically
            * compiled Go programs, always, even with cgo, using musl":
            * https://dominik.honnef.co/posts/2015/06/statically_compiled_go_programs__always__even_with_cgo__using_musl/
            */
            if (syscall(SYS_setns, nsref, namespaces[nsidx].nstype) < 0) {
                logerr("package gons: cannot join %s using reference \"%s\": %s", 
                    namespaces[nsidx].symname, nsenv,
                    strerror(errno));
                close(nsref);
                return;
            }
            /*
             * Release namespace reference fd, as by now our process should
             * reference the namespace by itself (unless there was an error),
             * and we don't want such open fds lying around anyway.
             */
            close(nsref);
        }
    }
}

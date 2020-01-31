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
    char *envvarname; /* name of env variable for this type of namespace */
    int   nstype;     /* CLONE_NEWxxx constant for this type of namespace. */
    char *path;       /* reference to a namespace in the currently mounted filesystem */
    int   fd;         /* optional fd referencing a namespace, if path==NULL */
};

/*
 * Defines the list of supported namespaces which can be set by gons before
 * the Go runtime spins up. Please note that setting the PID namespace will
 * never apply to us, but only to our children.
 */
static struct ns_t namespaces[] = {
    { "gons_cgroup", CLONE_NEWCGROUP, NULL, -1 },
    { "gons_ipc", CLONE_NEWIPC, NULL, -1 },
    { "gons_mnt", CLONE_NEWNS, NULL, -1 },
    { "gons_net", CLONE_NEWNET, NULL, -1 },
    { "gons_pid", CLONE_NEWPID, NULL, -1 },
    { "gons_user", CLONE_NEWUSER, NULL, -1 },
    { "gons_uts", CLONE_NEWUTS, NULL, -1 }
};

/* Number of namespace (types) */
#define NSCOUNT (sizeof(namespaces) / sizeof(namespaces[0]))

/* Default order if no order has been given ;) */
static char *defaultorder =
    "!user,!mnt,!cgroup,!ipc,!net,!pid,!uts";

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
    // Find out whether we should keep some ooooorder ;) The order describes
    // the sequence in which the namespaces should be entered whether the
    // paths are resolved into fds before the first setns(), or as the setns()
    // happen.
    int seq[NSCOUNT]; // indices into namespaces array
    int seqlen = 0;
    char *ooorder = getenv("gons_order");
    // In case no order has been given, then we will employ our default order.
    if (ooorder == NULL || !*ooorder) ooorder = defaultorder;
    // Remember: the environment is not ours ;) (...to write into)
    ooorder = strdup(ooorder);
    while (*ooorder && seqlen < NSCOUNT) {
        int fdref = *ooorder == '!';
        if (fdref) ++ooorder;
        char *delimiter = strchr(ooorder, ',');
        if (delimiter != NULL) *delimiter++ = '\0';
        // Find the corresponding type element in the namespaces array by name.
        // Please note that we skip the "gons_" prefix in the namespaces
        // definition above.
        int nsidx;
        for (nsidx = 0; nsidx < NSCOUNT; ++nsidx) {
            if (!strcmp(ooorder, namespaces[nsidx].envvarname+5)) {
                break;
            }
        }
        if (nsidx >= NSCOUNT) {
            logerr("package gons: unknown namespace type \"%s\" in gons_order",
                   ooorder);
            return;
        }
        // Get the corresponding filesystem path reference for this namespace.
        // If not set, then skip this sequence element.
        char *envvar = getenv(namespaces[nsidx].envvarname);
        if (envvar && *envvar) {
            // If the namespace should be entered using an fd-reference opened
            // before the first setns(), then open the fd now. Otherwise just
            // use the path later.
            if (fdref) {
                if (namespaces[nsidx].fd >= 0) {
                    logerr("package gons: duplicate namespace order type %s",
                           ooorder);
                    return;
                }
                int nsref = open(envvar, O_RDONLY);
                if (nsref < 0) {
                    logerr("package gons: invalid %s reference \"%s\": %s", 
                        namespaces[nsidx].envvarname, envvar,
                        strerror(errno));
                    return;
                }
                namespaces[nsidx].fd = nsref;
            }
            if (namespaces[nsidx].path) {
                logerr("package gons: duplicate namespace order type %s",
                       ooorder);
                return;
            }
            namespaces[nsidx].path = envvar;
            seq[seqlen] = nsidx;
            ++seqlen;
        }
        // If we had a delimiter, then it will by now already point past it,
        // thus to the next element in the sequence. If there wasn't a
        // delimiter, then we simple fast forward to the \0 after the last
        // element, so the loop will terminate.
        if (delimiter) {
            ooorder = delimiter;
        } else {
            ooorder += strlen(ooorder);
        }
    }
    // Now run through the namespace switch sequence and try to let things
    // happen...
    for (int seqidx = 0; seqidx < seqlen; ++seqidx) {
        int nsidx = seq[seqidx];
        int nsref = namespaces[nsidx].fd;
        // If there isn't a pre-opened fd for this namespace to switch into,
        // then we now need to open its reference.
        if (nsref < 0) {
            nsref = open(namespaces[nsidx].path, O_RDONLY);
            if (nsref < 0) {
                logerr("package gons: invalid %s reference \"%s\": %s", 
                    namespaces[nsidx].envvarname, namespaces[nsidx].path,
                    strerror(errno));
                return;
            }
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
        long res = syscall(SYS_setns, nsref, namespaces[nsidx].nstype);
        close(nsref); /* Don't leak file descriptors */
        if (res < 0) {
            logerr("package gons: cannot join %s using reference \"%s\": %s", 
                namespaces[nsidx].envvarname, namespaces[nsidx].path,
                strerror(errno));
            return;
        }
    }
}

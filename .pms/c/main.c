#include <dirent.h>
#include <errno.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <unistd.h>

void list_processes();
bool is_numeric(const char *str);
int kill_process(const char *pid_str, int signal);

void list_processes() {
  DIR *proc_dir = opendir("/proc");
  if (proc_dir == NULL) {
    perror("Error opening /proc");
    return;
  }
  struct dirent *entry;
  while ((entry = readdir(proc_dir)) != NULL) {
    if (is_numeric(entry->d_name)) {
      printf("%s\n", entry->d_name);
    }
  }
  closedir(proc_dir);
}

bool is_numeric(const char *str) {
  for (int i = 0; str[i] != '\0'; i++) {
    if (str[i] < '0' || str[i] > '9') {
      return false;
    }
  }
  return true;
}

int kill_process(const char *pid_str, int signal) {
  pid_t pid = (pid_t)atoi(pid_str);
  if (kill(pid, signal) == -1) {
    perror("Error sending signal");
    return errno;
  }
  return 0;
}

int main(int argc, char *argv[]) {
  if (argc < 2) {
    printf("Usage: %s <list|kill|help> [pid] [signal]\n", argv[0]);
    return EXIT_FAILURE;
  }
  if (strcmp(argv[1], "list") == 0) {
    list_processes();
  } else if (strcmp(argv[1], "kill") == 0) {
    if (argc < 4) {
      printf("Usage: %s kill <pid> <signal>\n", argv[0]);
      return EXIT_FAILURE;
    }
    if (!is_numeric(argv[2]) || !is_numeric(argv[3])) {
      printf("Error: pid and signal must be integers\n");
      return EXIT_FAILURE;
    }
    int signal = atoi(argv[3]);
    int result = kill_process(argv[2], signal);
    if (result != 0) {
      printf("Error: failed to send signal %d to process %s\n", signal,
             argv[2]);
      return EXIT_FAILURE;
    }
  } else if (strcmp(argv[1], "help") == 0) {
    printf("Usage: %s <list|kill|help> [pid] [signal]\n", argv[0]);
    printf("Commands:\n");
    printf("  list        List all running processes\n");
    printf("  kill <pid>  Kill process with specified PID\n");
    printf("  help        Show this help message\n");
  } else {
    printf("Error: unknown command '%s'\n", argv[1]);
    return EXIT_FAILURE;
  }
  return EXIT_SUCCESS;
}

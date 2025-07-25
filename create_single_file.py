import fnmatch
import os

# List of files and directories to ignore
ignore_list = [
    "switch-tube-downloader",
    "cmd",
    "*_test.go",
    ".*",
    "LICENSE",
    "README.md",
    "TODO.md",
    "combine_files.py",
    "combined_output.txt",
    "create_single_file.py",
    "go.mod",
    "go.sum",
]

# Output file
output_file = "combined_output.txt"

with open(output_file, "w", encoding="utf-8") as outfile:
    for root, dirs, files in os.walk("."):
        # Filter out ignored directories
        dirs[:] = [
            d
            for d in dirs
            if not any(fnmatch.fnmatch(d, pattern) for pattern in ignore_list)
        ]
        for file in files:
            if not any(fnmatch.fnmatch(file, pattern) for pattern in ignore_list):
                file_path = os.path.join(root, file)
                outfile.write(f"{file_path}:\n")
                try:
                    with open(file_path, "r", encoding="utf-8") as infile:
                        outfile.write(infile.read())
                except Exception:
                    outfile.write("[Error reading file]\n")
                outfile.write("\n\n")

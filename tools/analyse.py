import csv
from dataclasses import dataclass
import json
import os
from typing import List
from absl import app, logging, flags
import datetime
import pandas

flags.DEFINE_string("input", "store/output",
                    "The folder of .csv files to read.")

FLAGS = flags.FLAGS

VERSION_RELEASE_DATES = {
    "4.9": datetime.datetime(2022, 11, 15),
    # "4.8": datetime.datetime(2022, 11, 15),
    "4.7": datetime.datetime(2022, 5, 24),
    # "4.6": datetime.datetime(2022, 11, 15),
    "4.5": datetime.datetime(2021, 11, 17),
    "4.4": datetime.datetime(2021, 8, 26),
    "4.3": datetime.datetime(2021, 5, 26),
    "4.2": datetime.datetime(2021, 2, 23),
    "4.1": datetime.datetime(2020, 11, 19),
    "4.0": datetime.datetime(2020, 8, 20),
}

FEATURE_RELEASE_DATES = {
    "satisfies_expression": VERSION_RELEASE_DATES["4.9"],
    "accessor_keyword": VERSION_RELEASE_DATES["4.9"],
    "extends_constraint_on_infer": VERSION_RELEASE_DATES["4.7"],
    "variance_annotations_on_type_parameter": VERSION_RELEASE_DATES["4.7"],
    "type_modifier_on_import_name": VERSION_RELEASE_DATES["4.5"],
    "import_assertion": VERSION_RELEASE_DATES["4.5"],
    "static_block_in_class": VERSION_RELEASE_DATES["4.4"],
    "override_on_class_method": VERSION_RELEASE_DATES["4.3"],
    "abstract_construct_signature": VERSION_RELEASE_DATES["4.2"],
    "template_literal_type": VERSION_RELEASE_DATES["4.1"],
    "remapped_name_in_mapped_type": VERSION_RELEASE_DATES["4.1"],
    "named_tuple_member": VERSION_RELEASE_DATES["4.0"],
    "short_circuit_assignment": VERSION_RELEASE_DATES["4.0"],
}


def write_json(filename, obj):
    with open(filename, "w") as f:
        json.dump(obj, f)


@dataclass
class Row:
    id: str
    commit_hash: str
    pkg_name: str
    pkg_version: str
    ts_version: str
    total_files: int
    commit_time: datetime.datetime

    # feature flags
    satisfies_expression: bool
    accessor_keyword: bool
    extends_constraint_on_infer: bool
    variance_annotations_on_type_parameter: bool
    type_modifier_on_import_name: bool
    import_assertion: bool
    static_block_in_class: bool
    override_on_class_method: bool
    abstract_construct_signature: bool
    template_literal_type: bool
    remapped_name_in_mapped_type: bool
    named_tuple_member: bool
    short_circuit_assignment: bool

    def get_flags(self):
        return {
            "satisfies_expression": self.satisfies_expression,
            "accessor_keyword": self.accessor_keyword,
            "extends_constraint_on_infer": self.extends_constraint_on_infer,
            "variance_annotations_on_type_parameter": self.variance_annotations_on_type_parameter,
            "type_modifier_on_import_name": self.type_modifier_on_import_name,
            "import_assertion": self.import_assertion,
            "static_block_in_class": self.static_block_in_class,
            "override_on_class_method": self.override_on_class_method,
            "abstract_construct_signature": self.abstract_construct_signature,
            "template_literal_type": self.template_literal_type,
            "remapped_name_in_mapped_type": self.remapped_name_in_mapped_type,
            "named_tuple_member": self.named_tuple_member,
            "short_circuit_assignment": self.short_circuit_assignment,
        }


def b(s: str) -> bool:
    if s == "1":
        return True
    else:
        return False


def read_csv(filename) -> List[Row]:
    ret = []

    with open(filename) as f:
        r = csv.reader(f)
        for id, commit_hash, pkg_name, pkg_version, ts_version, commit_time, total_files, *rest in r:
            ret.append(Row(
                id, commit_hash, pkg_name, pkg_version, ts_version,
                int(total_files),
                datetime.datetime.fromtimestamp(int(commit_time)),
                *[b(i) for i in rest]
            ))

    return ret


def get_day_delta(a: datetime.datetime, b: datetime.datetime):
    return (a - b).days


def compute_deltas(flags: dict[str, bool], commit_time: datetime.datetime) -> dict[str, int | None]:
    ret = {}

    for k, v in flags.items():
        if v:
            release_date = FEATURE_RELEASE_DATES[k]
            delta_days = get_day_delta(commit_time, release_date)
            ret[k] = delta_days
        else:
            pass
            # ret[k] = None

    return ret


def update(d, key, value, default, compare_fn):
    if key not in d:
        d[key] = default
    d[key] = compare_fn(d[key], value)


def main(args):
    input_path = FLAGS.get_flag_value("input", "")

    earliest_usage = {}

    repo_feature_introduction: dict[str, dict[str, int]] = {}

    logging.info("Gathering basic statistics.")

    for file in os.listdir(input_path):
        filename = os.path.join(input_path, file)
        rows = read_csv(filename)
        for row in rows:
            row_flags = row.get_flags()

            flag_delta = compute_deltas(row_flags, row.commit_time)

            if row.id not in repo_feature_introduction:
                repo_feature_introduction[row.id] = {}
            repo = repo_feature_introduction[row.id]

            for k, v in flag_delta.items():
                if v == None:
                    continue
                if k not in repo:
                    repo[k] = v
                repo[k] = min(repo[k], v)

            update(repo, "total_files", row.total_files, 0, max)
            update(repo, "earliest_commit", row.commit_time.year, 3000, min)

            for k, v in row_flags.items():
                timestamp = row.commit_time.timestamp()
                if k not in earliest_usage:
                    earliest_usage[k] = (
                        "", datetime.datetime.now().timestamp())
                if v:
                    if timestamp < earliest_usage[k][1]:
                        earliest_usage[k] = (row.id, timestamp)

    logging.info("Writing basic statistics.")

    write_json("data/earliest_usage.json", earliest_usage)

    write_json("data/adoption_results.json", repo_feature_introduction)

    pandas.DataFrame(repo_feature_introduction) \
        .transpose() \
        .to_csv("data/adoption_results.csv")

    logging.info("Deriving milestone statistics.")

    milestone_stats = {
        "satisfies_expression": {},
        "accessor_keyword": {},
        "extends_constraint_on_infer": {},
        "variance_annotations_on_type_parameter": {},
        "type_modifier_on_import_name": {},
        "import_assertion": {},
        "static_block_in_class": {},
        "override_on_class_method": {},
        "abstract_construct_signature": {},
        "template_literal_type": {},
        "remapped_name_in_mapped_type": {},
        "named_tuple_member": {},
        "short_circuit_assignment": {},
    }

    for milestone in range(-50, 800, 10):
        for k in milestone_stats:
            milestone_stats[k][milestone] = 0

        for repo in repo_feature_introduction:
            for k in milestone_stats:
                adoption_time = repo_feature_introduction[repo].get(k, None)
                if adoption_time != None and adoption_time < milestone:
                    milestone_stats[k][milestone] += 1

    write_json("data/milestone_stats.json", milestone_stats)

    pandas.DataFrame(milestone_stats) \
        .to_csv("data/milestone_stats.csv")


if __name__ == "__main__":
    app.run(main)

import csv
from dataclasses import dataclass
import os
from typing import List
from absl import app, logging, flags
import datetime

flags.DEFINE_string("input", "store/output",
                    "The folder of .csv files to read.")

FLAGS = flags.FLAGS


@dataclass
class Row:
    id: str
    commit_hash: str
    pkg_name: str
    pkg_version: str
    ts_version: str
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
        for id, commit_hash, pkg_name, pkg_version, ts_version, commit_time, *rest in r:
            ret.append(Row(
                id, commit_hash, pkg_name, pkg_version, ts_version,
                datetime.datetime.fromtimestamp(int(commit_time)),
                *[b(i) for i in rest]
            ))

    return ret


def main(args):
    input_path = FLAGS.get_flag_value("input", "")

    earliest_usage = {}

    for file in os.listdir(input_path):
        filename = os.path.join(input_path, file)
        rows = read_csv(filename)
        for row in rows:
            row_flags = row.get_flags()

            for k, v in row_flags.items():
                if k not in earliest_usage:
                    earliest_usage[k] = ("", datetime.datetime.now())
                if v:
                    if row.commit_time < earliest_usage[k][1]:
                        earliest_usage[k] = (row.id, row.commit_time)

    logging.info("earliest = %s", earliest_usage)


if __name__ == "__main__":
    app.run(main)

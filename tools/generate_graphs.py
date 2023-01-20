import seaborn as sns
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
from matplotlib.colors import ListedColormap

plt.rcParams['text.usetex'] = True

my_cmap = ListedColormap(sns.color_palette(as_cmap=True))

milestone_stats = pd.read_csv("data/milestone_stats.csv")

version_df = milestone_stats[[
    "Unnamed: 0",
    "typescript_4.9",
    "typescript_4.7",
    "typescript_4.5",
    "typescript_4.4",
    "typescript_4.3",
    "typescript_4.2",
    "typescript_4.1",
    "typescript_4.0",
]]

version_df = version_df.rename(columns={
    "Unnamed: 0": "Days since Release",
    "typescript_4.9": "4.9",
    "typescript_4.7": "4.7",
    "typescript_4.5": "4.5",
    "typescript_4.4": "4.4",
    "typescript_4.3": "4.3",
    "typescript_4.2": "4.2",
    "typescript_4.1": "4.1",
    "typescript_4.0": "4.0",
})

feature_df = milestone_stats[[
    "Unnamed: 0",
    "satisfies_expression",
    "accessor_keyword",
    "extends_constraint_on_infer",
    "variance_annotations_on_type_parameter",
    "type_modifier_on_import_name",
    "import_assertion",
    "static_block_in_class",
    "override_on_class_method",
    "abstract_construct_signature",
    "template_literal_type",
    "remapped_name_in_mapped_type",
    "named_tuple_member",
    "short_circuit_assignment",
]]

feature_df = feature_df.rename(columns={
    "Unnamed: 0": "Days since Release",
    "satisfies_expression": "$f_0$ (4.9)",
    "accessor_keyword": "$f_1$ (4.9)",
    "extends_constraint_on_infer": "$f_2$ (4.7)",
    "variance_annotations_on_type_parameter": "$f_3$ (4.7)",
    "type_modifier_on_import_name": "$f_4$ (4.5)",
    "import_assertion": "$f_5$ (4.5)",
    "static_block_in_class": "$f_6$ (4.4)",
    "override_on_class_method": "$f_7$ (4.3)",
    "abstract_construct_signature": "$f_8$ (4.2)",
    "template_literal_type": "$f_9$ (4.1)",
    "remapped_name_in_mapped_type": "$f_{10}$ (4.1)",
    "named_tuple_member": "$f_{11}$ (4.0)",
    "short_circuit_assignment": "$f_{12}$ (4.0)",
})

"""
Version Adoption Graph
"""

fig, ax = plt.subplots(figsize=(5, 4))

version_df.plot(x="Days since Release", ax=ax, colormap=my_cmap)

ax.grid()
ax.set_ylabel("Repositories adopting version")
ax.set_xticks(np.arange(-100, 800, 100))

fig.savefig("graphs/typescript_version_adoption.pdf", bbox_inches='tight')

"""
Feature Adoption Graph
"""

fig, ax = plt.subplots(figsize=(12, 5))

feature_df.plot(x="Days since Release", ax=ax, colormap=my_cmap)

ax.grid()
ax.set_ylabel("Repositories adopting feature")
ax.set_xticks(np.arange(-100, 800, 100))

fig.savefig("graphs/typescript_feature_adoption.pdf", bbox_inches='tight')
